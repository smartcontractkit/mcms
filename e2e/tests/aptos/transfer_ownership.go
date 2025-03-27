//go:build e2e

package aptos

import (
	"context"
	"crypto/ecdsa"
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

func (a *AptosTestSuite) Test_Aptos_TransferOwnership() {
	a.deployMCMSContract()
	mcmsAddress := a.MCMSContract.Address()
	opts := &bind.TransactOpts{Signer: a.deployerAccount}

	// Set config on contract
	signers := [2]common.Address{}
	signerKeys := [2]*ecdsa.PrivateKey{}
	for i := range signers {
		signerKeys[i], _ = crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(signerKeys[i].PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})
	a.T().Logf("Signers: %v", signers)
	config := &types.Config{
		Quorum:  2,
		Signers: []common.Address{signers[0], signers[1]},
	}
	configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount)
	result, err := configurer.SetConfig(context.Background(), mcmsAddress.StringLong(), config, false)
	a.Require().NoError(err, "setting config on Aptos mcms contract failed")

	data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("✅ SetConfig in tx: %s", result.Hash)

	// Initiate ownership transfer
	tx, err := a.MCMSContract.MCMSAccount().TransferOwnershipToSelf(opts)
	a.Require().NoError(err)
	data, err = a.AptosRPCClient.WaitForTransaction(tx.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("🚀 TransferOwnershipToSelf in tx: %s", tx.Hash)

	// Build proposal
	validUntil := time.Now().Add(time.Hour * 24).Unix()
	proposalBuilder := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil)).
		SetDescription("Test accepting ownership of the contract itself").
		SetOverridePreviousRoot(true).
		AddChainMetadata(a.ChainSelector, types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      mcmsAddress.StringLong(),
		})

	// Call 1
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
	proposalBuilder.AddOperation(types.Operation{
		ChainSelector: a.ChainSelector,
		Transaction:   transaction,
	})

	proposal, err := proposalBuilder.Build()
	a.Require().NoError(err, "Error building proposal")

	// Sign proposal
	inspector := aptossdk.NewInspector(a.AptosRPCClient)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		a.ChainSelector: inspector,
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	a.Require().NoError(err, "Error creating signable")
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKeys[0]))
	a.Require().NoError(err, "Error signing with key 0")
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerKeys[1]))
	a.Require().NoError(err, "Error signing with key 1")

	// Validate signatures
	quorumMet, err := signable.ValidateSignatures(context.Background())
	a.Require().NoError(err, "Error validating signatures")
	a.Require().True(quorumMet, "Quorum not met")

	// Set Root
	encoders, err := proposal.GetEncoders()
	a.Require().NoError(err)
	aptosEncoder := encoders[a.ChainSelector].(*aptossdk.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{
		a.ChainSelector: aptossdk.NewExecutor(a.AptosRPCClient, a.deployerAccount, aptosEncoder),
	}
	executable, err := mcms.NewExecutable(proposal, executors)
	a.Require().NoError(err, "Error creating executable")

	txHash, err := executable.SetRoot(context.Background(), a.ChainSelector)
	a.Require().NoError(err)
	a.T().Logf("✅ SetRoot in tx: %s", txHash.Hash)

	data, err = a.AptosRPCClient.WaitForTransaction(txHash.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)

	// Assert
	tree, _ := proposal.MerkleTree()
	gotHash, gotValidUntil, err := inspector.GetRoot(context.Background(), mcmsAddress.StringLong())
	a.Require().NoError(err)
	a.Require().Equal(uint32(validUntil), gotValidUntil)
	a.Require().Equal(tree.Root, gotHash)

	// Execute
	for i := range proposal.Operations {
		a.T().Logf("Executing operation: %v", i)
		var txOutput types.TransactionResult
		txOutput, err = executable.Execute(context.Background(), i)
		a.Require().NoError(err)
		a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)

		data, err = a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)

		// Assert

		// Check that op count has increased on the mcms contract
		var opCount uint64
		opCount, err = inspector.GetOpCount(context.Background(), mcmsAddress.StringLong())
		a.Require().NoError(err)
		a.Require().EqualValues(opCount, i+1)
	}

	// Check that ownership has been transferred
	owner, err := a.MCMSContract.MCMSAccount().Owner(nil)
	a.Require().NoError(err)
	a.Require().Equal(mcmsAddress.StringLong(), owner.StringLong())

	a.T().Logf("MCMS contract owner: %v", owner.StringLong())
}
