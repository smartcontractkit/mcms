//go:build e2e

package aptos

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	module_mcms_user "github.com/smartcontractkit/chainlink-aptos/bindings/mcms_test/mcms_user"
	"github.com/smartcontractkit/chainlink-aptos/relayer/codec"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *AptosTestSuite) Test_Aptos_TimelockProposal() {
	/*
		This tests that both the Proposers and the Bypassers can successfully perform operations
		via the timelock.

		1. Configure the Bypasser signers
		2. Configure the Proposer signers
		3. Initiate the ownership transfer from the deployer EOA (transfer_ownership)
		4. Create a Proposer timelock proposal to accept ownership of itself (accept_ownership)
			a. Set root and execute the proposal (schedule_batch)
			b. Wait for the proposal to become ready (should take at least 10 seconds)
			c. After it is ready, execute the timelock operation (execute_batch)
			d. Check that ownership has actually been transferred
		5. Create a Bypasser timelock proposal to call a function on the MCMSTest contract
			a. Set root and execute the proposal (bypasser_execute_batch)
			b. Check that the target contract has actually been called
	*/
	a.deployMCMSContract()
	a.deployMCMSTestContract()
	mcmsAddress := a.MCMSContract.Address()
	mcmsTestAddress := a.MCMSTestContract.Address()
	opts := &bind.TransactOpts{Signer: a.deployerAccount}

	// Configure Bypassers
	bypassers := [2]common.Address{}
	bypasserKeys := [2]*ecdsa.PrivateKey{}
	for i := range bypassers {
		bypasserKeys[i], _ = crypto.GenerateKey()
		bypassers[i] = crypto.PubkeyToAddress(bypasserKeys[i].PublicKey)
	}
	slices.SortFunc(bypassers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})
	{
		bypasserConfig := &types.Config{
			Quorum:  2,
			Signers: bypassers[:],
		}
		bypassConfigurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleBypasser)
		result, err := bypassConfigurer.SetConfig(a.T().Context(), mcmsAddress.StringLong(), bypasserConfig, false)
		a.Require().NoError(err)
		data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
	}
	// Get Min Delay
	timelockInspector := aptossdk.NewTimelockInspector(a.AptosRPCClient)
	delay, err := timelockInspector.GetMinDelay(a.T().Context(), mcmsAddress.StringLong())
	a.Require().NoError(err)
	a.Require().Equal(102, delay)
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

	// ====================================================
	// | First proposal - accept ownership with proposers |
	// ====================================================

	{
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
			SetDelay(types.NewDuration(time.Second * 2))

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

		inspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleProposer)
		inspectorsMap := map[types.ChainSelector]sdk.Inspector{
			a.ChainSelector: inspector,
		}
		signable, err := mcms.NewSignable(&acceptOwnershipProposal, inspectorsMap)
		a.Require().NoError(err)

		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKeys[0]))
		a.Require().NoError(err)
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKeys[1]))
		a.Require().NoError(err)
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKeys[2]))
		a.Require().NoError(err)

		quorumMet, err := signable.ValidateSignatures(a.T().Context())
		a.Require().NoError(err, "Error validating signatures")
		a.Require().True(quorumMet, "Quorum not met")

		// Set Root
		encoders, err := acceptOwnershipProposal.GetEncoders()
		a.Require().NoError(err)
		aptosEncoder := encoders[a.ChainSelector].(*aptossdk.Encoder)
		executors := map[types.ChainSelector]sdk.Executor{
			a.ChainSelector: aptossdk.NewExecutor(a.AptosRPCClient, a.deployerAccount, aptosEncoder, aptossdk.TimelockRoleProposer),
		}
		executable, err := mcms.NewExecutable(&acceptOwnershipProposal, executors)
		a.Require().NoError(err, "Error creating executable")

		result, err := executable.SetRoot(a.T().Context(), a.ChainSelector)
		a.Require().NoError(err)

		data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
		a.Require().NoError(err)
		a.Require().True(data.Success, data.VmStatus)
		a.T().Logf("✅ SetRoot in tx: %s", result.Hash)

		// Assert
		tree, _ := acceptOwnershipProposal.MerkleTree()
		gotHash, gotValidUntil, err := inspector.GetRoot(a.T().Context(), mcmsAddress.StringLong())
		a.Require().NoError(err)
		a.Require().Equal(validUntil, gotValidUntil)
		a.Require().Equal(tree.Root, gotHash)

		// Execute
		start := time.Now()
		for i := range acceptOwnershipProposal.Operations {
			a.T().Logf("Executing operation: %v", i)
			txOutput, xerr := executable.Execute(a.T().Context(), i)
			a.Require().NoError(xerr)
			data, err = a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
			a.Require().NoError(err)
			a.Require().True(data.Success, data.VmStatus)
			a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)

			// Assert

			// Check that op count has increased on the mcms contract
			var opCount uint64
			opCount, err = inspector.GetOpCount(a.T().Context(), mcmsAddress.StringLong())
			a.Require().NoError(err)
			a.Require().EqualValues(opCount, i+1)
		}

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

		a.Require().EventuallyWithT(
			func(collect *assert.CollectT) {
				assert.NoErrorf(collect, timelockExecutable.IsReady(a.T().Context()), "Proposal is not ready")
			},
			time.Second*4,
			time.Millisecond*500,
		)
		elapsed := time.Since(start)
		a.Require().Greaterf(elapsed, time.Second*2, "Proposal should only be ready after 2 seconds")
		a.T().Logf("🟢 Timelock operation ready, elapsed time: %v", elapsed.String())

		for i := range acceptOwnershipTimelockProposal.Operations {
			res, err := timelockExecutable.Execute(a.T().Context(), i)
			a.Require().NoError(err)
			data, err = a.AptosRPCClient.WaitForTransaction(res.Hash)
			a.Require().NoError(err)
			a.Require().True(data.Success, data.VmStatus)
			a.T().Logf("Timelock operation %v executed", i)
		}
	}

	owner, err := a.MCMSContract.MCMSAccount().Owner(nil)
	a.Require().NoError(err)
	a.Require().Equal(mcmsAddress.StringLong(), owner.StringLong())
	a.T().Logf("MCMS Ownership transferred to itself")

	// ==============================================
	// | Second proposal - test mcms with bypassers |
	// ==============================================

	// Arguments to call MCMUser contract with
	arg1 := "helloworld"
	arg2 := []byte{5, 4, 3, 2, 1}
	arg3 := a.deployerAccount.AccountAddress()
	arg4 := big.NewInt(42)

	{
		inspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleBypasser)
		inspectorsMaps := map[types.ChainSelector]sdk.Inspector{
			a.ChainSelector: inspector,
		}

		startingOpCount, errr := inspector.GetOpCount(a.T().Context(), mcmsAddress.StringLong())
		a.Require().NoError(errr)
		validUntil := uint32(time.Now().Add(time.Hour * 24).Unix())
		mcmsTestProposalBuilder := mcms.NewTimelockProposalBuilder().
			SetVersion("v1").
			SetValidUntil(validUntil).
			SetDescription("Test bypasser proposal").
			AddTimelockAddress(a.ChainSelector, mcmsAddress.StringLong()).
			AddChainMetadata(a.ChainSelector, types.ChainMetadata{
				StartingOpCount:  startingOpCount,
				MCMAddress:       mcmsAddress.StringLong(),
				AdditionalFields: Must(json.Marshal(aptossdk.AdditionalFieldsMetadata{Role: aptossdk.TimelockRoleBypasser})),
			}).
			SetAction(types.TimelockActionBypass)

		// Call 1
		module, function, _, args, errr := a.MCMSTestContract.MCMSUser().Encoder().FunctionOne(arg1, arg2)
		a.Require().NoError(errr)
		bop := types.BatchOperation{
			ChainSelector: a.ChainSelector,
		}
		tx, errr := aptossdk.NewTransaction(
			module.PackageName,
			module.ModuleName,
			function,
			a.MCMSTestContract.Address(),
			aptossdk.ArgsToData(args),
			"MCMSTest",
			nil,
		)
		a.Require().NoError(errr)
		bop.Transactions = append(bop.Transactions, tx)

		// Call 2
		module, function, _, args, err = a.MCMSTestContract.MCMSUser().Encoder().FunctionTwo(arg3, arg4)
		a.Require().NoError(err)
		tx, err = aptossdk.NewTransaction(
			module.PackageName,
			module.ModuleName,
			function,
			a.MCMSTestContract.Address(),
			aptossdk.ArgsToData(args),
			"MCMSTest",
			nil,
		)
		a.Require().NoError(err)
		bop.Transactions = append(bop.Transactions, tx)

		mcmsTestProposalBuilder.AddOperation(bop)

		mcmsTestTimelockProposal, errr := mcmsTestProposalBuilder.Build()
		a.Require().NoError(errr)

		convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
			a.ChainSelector: aptossdk.NewTimelockConverter(),
		}
		mcmsTestProposal, _, errr := mcmsTestTimelockProposal.Convert(a.T().Context(), convertersMap)
		a.Require().NoError(errr)

		signable, errr := mcms.NewSignable(&mcmsTestProposal, inspectorsMaps)
		a.Require().NoError(errr)

		_, errr = signable.SignAndAppend(mcms.NewPrivateKeySigner(bypasserKeys[0]))
		a.Require().NoError(errr)
		_, errr = signable.SignAndAppend(mcms.NewPrivateKeySigner(bypasserKeys[1]))
		a.Require().NoError(errr)

		quorumMet, errr := signable.ValidateSignatures(a.T().Context())
		a.Require().NoError(errr)
		a.Require().True(quorumMet, "Quorum not met")

		// Set Root
		encoders, errr := mcmsTestProposal.GetEncoders()
		a.Require().NoError(errr)
		aptosEncoder := encoders[a.ChainSelector].(*aptossdk.Encoder)
		executors := map[types.ChainSelector]sdk.Executor{
			a.ChainSelector: aptossdk.NewExecutor(a.AptosRPCClient, a.deployerAccount, aptosEncoder, aptossdk.TimelockRoleBypasser),
		}
		executable, errr := mcms.NewExecutable(&mcmsTestProposal, executors)
		a.Require().NoError(errr)

		result, errr := executable.SetRoot(a.T().Context(), a.ChainSelector)
		a.Require().NoError(errr)

		data, errr := a.AptosRPCClient.WaitForTransaction(result.Hash)
		a.Require().NoError(errr)
		a.Require().True(data.Success, data.VmStatus)
		a.T().Logf("✅ SetRoot in tx: %s", result.Hash)

		// Execute
		for i := range mcmsTestProposal.Operations {
			a.T().Logf("Executing operation: %v", i)
			txOutput, errr := executable.Execute(a.T().Context(), i)
			a.Require().NoError(errr)
			data, errr := a.AptosRPCClient.WaitForTransaction(txOutput.Hash)
			a.Require().NoError(errr)
			a.Require().True(data.Success, data.VmStatus)
			a.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)
		}
	}

	// Check that arguments have been stored in the MCMUser contract
	resourceData, err := a.AptosRPCClient.AccountResource(mcmsTestAddress, mcmsTestAddress.StringLong()+"::mcms_user::UserData")
	a.Require().NoError(err)

	userData := module_mcms_user.UserData{}
	err = codec.DecodeAptosJsonValue(resourceData["data"], &userData)
	a.Require().NoError(err)
	a.Require().EqualValues(2, userData.Invocations)
	a.Require().Equal(arg1, userData.A)
	a.Require().Equal(arg2, userData.B)
	a.Require().Equal(arg3, userData.C)
	a.Require().Equal(arg4, userData.D)
}
