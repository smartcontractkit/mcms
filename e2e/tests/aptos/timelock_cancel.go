//go:build e2e

package aptos

import (
	"crypto/ecdsa"
	"encoding/json"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_TimelockCancel() {
	/*
		This tests that a timelock proposal scheduled by the Proposer MCM can be cancelled by the
		Canceller MCM.

		1. Configure the Canceller signers
		2. Configure the Proposer signers
		3. Initiate the ownership transfer from the deployer EOA (transfer_ownership)
		4. Create and schedule a proposal using the Proposer MCM to accept ownership
		5. Check that the operation has actually been scheduled with the timelock (is_operation)
		6. Derive a cancellation-proposal form the timelock proposal and execute it
		7. Check that the operation has actually been cancelled -> is_operation should return false
	*/
	a.deployMCMSContract()
	mcmsAddress := a.MCMSContract.Address()
	opts := &bind.TransactOpts{Signer: a.deployerAccount}

	// Configure Cancellers
	cancellers := [2]common.Address{}
	cancellerKeys := [2]*ecdsa.PrivateKey{}
	for i := range cancellers {
		cancellerKeys[i], _ = crypto.GenerateKey()
		cancellers[i] = crypto.PubkeyToAddress(cancellerKeys[i].PublicKey)
	}
	slices.SortFunc(cancellers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})
	{
		cancellerConfig := &types.Config{
			Quorum:  2,
			Signers: cancellers[:],
		}
		cancelerConfigurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleCanceller)
		result, err := cancelerConfigurer.SetConfig(a.T().Context(), mcmsAddress.StringLong(), cancellerConfig, false)
		a.Require().NoError(err)
		data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
	}

	// Configure Proposers
	proposers := [3]common.Address{}
	proposerKeys := [3]*ecdsa.PrivateKey{}
	for i := range proposers {
		proposerKeys[i], _ = crypto.GenerateKey()
		proposers[i] = crypto.PubkeyToAddress(proposerKeys[i].PublicKey)
	}
	slices.SortFunc(proposers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})
	{
		proposerConfig := &types.Config{
			Quorum:  3,
			Signers: proposers[:],
		}
		proposeConfigurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleProposer)
		result, err := proposeConfigurer.SetConfig(a.T().Context(), mcmsAddress.StringLong(), proposerConfig, false)
		a.Require().NoError(err)
		data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
	}

	// Initiate ownership transfer
	{
		tx, err := a.MCMSContract.MCMSAccount().TransferOwnershipToSelf(opts)
		a.Require().NoError(err)
		data, err := a.AptosRPCClient.WaitForTransaction(tx.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
		a.T().Logf("🚀 TransferOwnershipToSelf in tx: %s", tx.Hash)
	}

	// =======================================================
	// | Proposal - schedule accept ownership with proposers |
	// =======================================================

	validUntil := uint32(time.Now().Add(time.Hour * 24).Unix())
	acceptOwnershipProposalBuilder := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Accept ownership via timelock").
		AddTimelockAddress(a.ChainSelector, mcmsAddress.StringLong()).
		AddChainMetadata(a.ChainSelector, types.ChainMetadata{
			StartingOpCount:  0,
			MCMAddress:       mcmsAddress.StringLong(),
			AdditionalFields: Must(json.Marshal(aptossdk.AdditionalFieldsMetadata{Role: aptossdk.TimelockRoleProposer})),
		}).
		SetAction(types.TimelockActionSchedule).
		SetDelay(types.NewDuration(time.Second))

	module, function, _, args, err := a.MCMSContract.MCMSAccount().Encoder().AcceptOwnership()
	a.Require().NoError(err)
	transaction, err := aptossdk.NewTransaction(
		module.PackageName,
		module.ModuleName,
		function,
		a.MCMSContract.Address(),
		aptossdk.ArgsToData(args),
		"MCMS",
		nil,
	)
	a.Require().NoError(err)
	acceptOwnershipProposalBuilder.AddOperation(types.BatchOperation{
		ChainSelector: a.ChainSelector,
		Transactions:  []types.Transaction{transaction},
	})
	acceptOwnershipTimelockProposal, err := acceptOwnershipProposalBuilder.Build()
	a.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		a.ChainSelector: aptossdk.NewTimelockConverter(),
	}
	acceptOwnershipProposal, _, err := acceptOwnershipTimelockProposal.Convert(a.T().Context(), convertersMap)
	a.Require().NoError(err)

	proposerInspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleProposer)
	proposerInspectorsMap := map[types.ChainSelector]sdk.Inspector{
		a.ChainSelector: proposerInspector,
	}
	proposerSignable, err := mcms.NewSignable(&acceptOwnershipProposal, proposerInspectorsMap)
	a.Require().NoError(err)

	_, err = proposerSignable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKeys[0]))
	a.Require().NoError(err)
	_, err = proposerSignable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKeys[1]))
	a.Require().NoError(err)
	_, err = proposerSignable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKeys[2]))
	a.Require().NoError(err)

	quorumMet, err := proposerSignable.ValidateSignatures(a.T().Context())
	a.Require().NoError(err, "Error validating signatures")
	a.Require().True(quorumMet, "Quorum not met")

	// Set Root
	encoders, err := acceptOwnershipProposal.GetEncoders()
	a.Require().NoError(err)
	aptosEncoder := encoders[a.ChainSelector].(*aptossdk.Encoder)
	proposerExecutors := map[types.ChainSelector]sdk.Executor{
		a.ChainSelector: aptossdk.NewExecutor(a.AptosRPCClient, a.deployerAccount, aptosEncoder, aptossdk.TimelockRoleProposer),
	}
	proposerExecutable, err := mcms.NewExecutable(&acceptOwnershipProposal, proposerExecutors)
	a.Require().NoError(err, "Error creating executable")

	result, err := proposerExecutable.SetRoot(a.T().Context(), a.ChainSelector)
	a.Require().NoError(err)

	data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("✅ SetRoot in tx: %s", result.Hash)

	// Assert
	tree, _ := acceptOwnershipProposal.MerkleTree()
	gotHash, gotValidUntil, err := proposerInspector.GetRoot(a.T().Context(), mcmsAddress.StringLong())
	a.Require().NoError(err)
	a.Require().Equal(validUntil, gotValidUntil)
	a.Require().Equal(tree.Root, gotHash)

	// Execute
	a.T().Logf("Executing operation: %v", 0)
	txOutput, err := proposerExecutable.Execute(a.T().Context(), 0)
	a.Require().NoError(err)
	data, err = a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)

	// Assert

	// Check that op count has increased on the mcms contract
	var opCount uint64
	opCount, err = proposerInspector.GetOpCount(a.T().Context(), mcmsAddress.StringLong())
	a.Require().NoError(err)
	a.Require().EqualValues(opCount, 1)

	timelockExecutor := aptossdk.NewTimelockExecutor(a.AptosRPCClient, a.deployerAccount)
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		a.ChainSelector: timelockExecutor,
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(a.T().Context(), acceptOwnershipTimelockProposal, timelockExecutors)
	a.Require().NoError(err)

	operationID, err := timelockExecutable.GetOpID(a.T().Context(), 0, acceptOwnershipTimelockProposal.Operations[0], a.ChainSelector)
	a.Require().NoError(err)
	timelockInspector := aptossdk.NewTimelockInspector(a.AptosRPCClient)
	ok, err := timelockInspector.IsOperation(a.T().Context(), mcmsAddress.StringLong(), operationID)
	a.Require().NoError(err)
	a.Require().True(ok, "Operation not found in timelock")

	// ======================================================
	// | Proposal - cancel accept ownership with cancellers |
	// ======================================================

	cancelTimelockProposal, err := acceptOwnershipTimelockProposal.DeriveCancellationProposal(map[types.ChainSelector]types.ChainMetadata{
		a.ChainSelector: {
			StartingOpCount:  0,
			MCMAddress:       mcmsAddress.StringLong(),
			AdditionalFields: Must(json.Marshal(aptossdk.AdditionalFieldsMetadata{Role: aptossdk.TimelockRoleCanceller})),
		},
	})
	a.Require().NoError(err)

	cancelProposal, _, err := cancelTimelockProposal.Convert(a.T().Context(), convertersMap)
	a.Require().NoError(err)

	cancellerInspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleCanceller)
	cancellerInspectorMap := map[types.ChainSelector]sdk.Inspector{
		a.ChainSelector: cancellerInspector,
	}
	cancellerSignable, err := mcms.NewSignable(&cancelProposal, cancellerInspectorMap)
	a.Require().NoError(err)

	_, err = cancellerSignable.SignAndAppend(mcms.NewPrivateKeySigner(cancellerKeys[0]))
	a.Require().NoError(err)
	_, err = cancellerSignable.SignAndAppend(mcms.NewPrivateKeySigner(cancellerKeys[1]))
	a.Require().NoError(err)

	quorumMet, err = cancellerSignable.ValidateSignatures(a.T().Context())
	a.Require().NoError(err)
	a.Require().True(quorumMet, "Quorum not met")

	// Set Root
	cancellerExecutors := map[types.ChainSelector]sdk.Executor{
		a.ChainSelector: aptossdk.NewExecutor(a.AptosRPCClient, a.deployerAccount, aptosEncoder, aptossdk.TimelockRoleCanceller),
	}
	cancelExecutable, err := mcms.NewExecutable(&cancelProposal, cancellerExecutors)
	a.Require().NoError(err)

	result, err = cancelExecutable.SetRoot(a.T().Context(), a.ChainSelector)
	a.Require().NoError(err)
	data, err = a.AptosRPCClient.WaitForTransaction(result.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("SetRoot of cancel proposal: %s", result.Hash)

	txOutput, err = cancelExecutable.Execute(a.T().Context(), 0)
	a.Require().NoError(err)
	data, err = a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("Canceled operation")

	// Assert
	ok, err = timelockInspector.IsOperation(a.T().Context(), mcmsAddress.StringLong(), operationID)
	a.Require().NoError(err)
	a.Require().False(ok, "Operation was cancelled but is still found in timelock")

	a.T().Logf("🔓 Timelock operation %v canceled successfully", operationID)
}
