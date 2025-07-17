//go:build e2e

package sui

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func (s *SuiTestSuite) Test_Sui_TimelockProposal() {
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
	bypasserConfig := &types.Config{
		Quorum:  2,
		Signers: bypassers[:],
	}

	proposers := [3]common.Address{}
	proposerKeys := [3]*ecdsa.PrivateKey{}
	for i := range proposers {
		proposerKeys[i], _ = crypto.GenerateKey()
		proposers[i] = crypto.PubkeyToAddress(proposerKeys[i].PublicKey)
	}
	slices.SortFunc(proposers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	proposerConfig := &types.Config{
		Quorum:  2,
		Signers: proposers[:],
	}

	fmt.Println("Bypassers:", bypassers)
	fmt.Println("Proposers:", proposers)

	// Set config
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleBypasser, s.mcmsPackageId, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcmsObj, bypasserConfig, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageId, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcmsObj, proposerConfig, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}

	// Init transfer ownership to self
	{
		tx, err := s.mcmsAccount.TransferOwnershipToSelf(
			s.T().Context(),
			&bind.CallOpts{
				Signer:           s.signer,
				WaitForExecution: true,
			},
			bind.Object{Id: s.ownerCapObj},
			bind.Object{Id: s.accountObj},
		)
		s.Require().NoError(err, "Failed to transfer ownership to self")
		s.Require().NotEmpty(tx, "Transaction should not be empty")
	}

	var timelockProposal *mcms.TimelockProposal

	// Create a timelock proposal accepting the ownership transfer
	{

		// Get the accept ownership call information and build the MCMS Operation
		encodedCall, err := s.mcmsAccount.Encoder().AcceptOwnershipAsTimelock(bind.Object{Id: s.accountObj})
		s.Require().NoError(err)

		callBytes := []byte{}
		if len(encodedCall.CallArgs) > 0 && encodedCall.CallArgs[0].CallArg.Pure != nil {
			callBytes = encodedCall.CallArgs[0].CallArg.Pure.Bytes
		}

		transaction, err := suisdk.NewTransaction(
			encodedCall.Module.ModuleName,
			encodedCall.Function,
			encodedCall.Module.PackageID,
			callBytes,
			"MCMS",
			[]string{},
		)
		s.Require().NoError(err)
		op := types.BatchOperation{
			ChainSelector: s.chainSelector,
			Transactions:  []types.Transaction{transaction},
		}

		// Construct the timelock proposal
		validUntilMs := uint32(time.Now().Add(time.Hour * 24).Unix())
		acceptOwnershipProposalBuilder := mcms.NewTimelockProposalBuilder().
			SetVersion("v1").
			SetValidUntil(validUntilMs).
			SetDescription("Accept ownership via timelock").
			AddTimelockAddress(s.chainSelector, s.mcmsObj).
			AddChainMetadata(s.chainSelector, types.ChainMetadata{
				StartingOpCount:  0,
				MCMAddress:       s.mcmsObj,
				AdditionalFields: Must(json.Marshal(suisdk.AdditionalFieldsMetadata{Role: suisdk.TimelockRoleBypasser})),
			}).
			SetAction(types.TimelockActionBypass).
			AddOperation(op)

		timelockProposal, err = acceptOwnershipProposalBuilder.Build()
		s.Require().NoError(err)
	}

	// Sign the proposal, set root and execute proposal operations
	{

		// Convert the Timelock Proposal into a MCMS Proposal
		timelockConverter, err := suisdk.NewTimelockConverter(s.client, s.signer, s.mcmsPackageId)
		s.Require().NoError(err)

		convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
			s.chainSelector: timelockConverter,
		}
		proposal, _, err := timelockProposal.Convert(s.T().Context(), convertersMap)
		s.Require().NoError(err)

		// TODO: Remove
		proposalJSON, err := json.MarshalIndent(proposal, "", "  ")
		s.Require().NoError(err)
		s.T().Logf("Proposal JSON:\n%s", string(proposalJSON))

		inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleBypasser)
		s.Require().NoError(err, "creating inspector for Sui mcms contract")
		inspectorsMap := map[types.ChainSelector]sdk.Inspector{
			s.chainSelector: inspector,
		}

		s.T().Logf("Signing the proposal...")
		// Sign the proposal with the bypasser keys
		signable, err := mcms.NewSignable(&proposal, inspectorsMap)
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(bypasserKeys[0]))
		s.Require().NoError(err)
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(bypasserKeys[1]))
		s.Require().NoError(err)

		quorumMet, err := signable.ValidateSignatures(s.T().Context())
		s.Require().NoError(err, "Error validating signatures")
		s.Require().True(quorumMet, "Quorum not met")

		// Set Root
		s.T().Logf("Preparing to the root of the proposal...")
		encoders, err := proposal.GetEncoders()
		s.Require().NoError(err)
		suiEncoder := encoders[s.chainSelector].(*suisdk.Encoder)
		executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.mcmsPackageId, suisdk.TimelockRoleBypasser, s.accountObj, s.registryObj)
		s.Require().NoError(err, "creating executor for Sui mcms contract")
		executors := map[types.ChainSelector]sdk.Executor{
			s.chainSelector: executor,
		}
		executable, err := mcms.NewExecutable(&proposal, executors)
		s.Require().NoError(err, "Error creating executable")

		s.T().Logf("Setting the root of the proposal...")

		result, err := executable.SetRoot(s.T().Context(), s.chainSelector)
		s.Require().NoError(err)

		s.T().Logf("✅ SetRoot in tx: %s", result.Hash)

		s.T().Logf("Executing the proposal operations...")
		// Execute
		for i := range proposal.Operations {
			s.T().Logf("Executing operation: %v", i)
			txOutput, err := executable.Execute(s.T().Context(), i)
			s.Require().NoError(err)
			s.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)
		}

		// Complete the proposal transfer
		s.T().Logf("Completing the proposal transfer...")
		tx, err := s.mcmsAccount.ExecuteOwnershipTransfer(s.T().Context(), &bind.CallOpts{
			Signer:           s.signer,
			WaitForExecution: true,
		}, bind.Object{Id: s.ownerCapObj}, bind.Object{Id: s.accountObj}, bind.Object{Id: s.registryObj}, "0x0")
		s.Require().NoError(err, "Failed to execute ownership transfer")
		s.Require().NotEmpty(tx, "Transaction should not be empty")
		s.T().Logf("✅ Executed ownership transfer in tx: %s", tx.Digest)

		// Check owner
		owner, err := bind.ReadObject(s.T().Context(), s.accountObj, s.client)
		s.Require().NoError(err)
		// TODO: Due to the @mcms problem, the owner is set to zero instead of the actual mcms package ID
		s.Require().Equal("0x0000000000000000000000000000000000000000000000000000000000000000", owner.Data.Content.Fields["owner"], "Owner should be the owner cap object")

	}

}
