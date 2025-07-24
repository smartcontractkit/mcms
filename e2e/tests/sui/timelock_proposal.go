//go:build e2e

package sui

import (
	"crypto/ecdsa"
	"encoding/json"
	"slices"
	"testing"
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
	s.T().Run("TimelockProposal - MCMSAccount Accept Ownership through Bypass", func(t *testing.T) {
		RunAcceptOwnershipProposal(s, suisdk.TimelockRoleBypasser)
	})

	s.T().Run("TimelockProposal - MCMSAccount Accept Ownership through Schedule", func(t *testing.T) {
		RunAcceptOwnershipProposal(s, suisdk.TimelockRoleProposer)
	})
}

func RunAcceptOwnershipProposal(s *SuiTestSuite, role suisdk.TimelockRole) {
	s.T().Logf("Running accept ownership proposal with role: %v", role)
	s.DeployMCMSContract()

	// Init transfer ownership
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

	delay_1_min := time.Minute * 1
	// Create a timelock proposal accepting the ownership transfer

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
		AddTimelockAddress(s.chainSelector, s.timelockObj).
		AddChainMetadata(s.chainSelector, types.ChainMetadata{
			StartingOpCount:  0,
			MCMAddress:       s.mcmsObj,
			AdditionalFields: Must(json.Marshal(suisdk.AdditionalFieldsMetadata{Role: role})),
		}).
		AddOperation(op)
	// Set the action based on the role
	if role == suisdk.TimelockRoleProposer {
		acceptOwnershipProposalBuilder.
			SetAction(types.TimelockActionSchedule).
			SetDelay(types.NewDuration(delay_1_min))
	} else if role == suisdk.TimelockRoleBypasser {
		// If bypasser, we need to set the action to accept ownership
		acceptOwnershipProposalBuilder.SetAction(types.TimelockActionBypass)
	} else {
		s.T().Fatalf("Unsupported role: %v", role)
	}

	timelockProposal, err = acceptOwnershipProposalBuilder.Build()
	s.Require().NoError(err)

	// Sign the proposal, set root and execute proposal operations

	// Convert the Timelock Proposal into a MCMS Proposal
	timelockConverter, err := suisdk.NewTimelockConverter(s.client, s.signer, s.mcmsPackageId)
	s.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(s.T().Context(), convertersMap)
	s.Require().NoError(err)

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, role)
	s.Require().NoError(err, "creating inspector for Sui mcms contract")
	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}

	s.T().Logf("Signing the proposal...")
	// Sign the proposal with the corresponding keys
	var keys []*ecdsa.PrivateKey
	if role == suisdk.TimelockRoleProposer {
		keys = proposerKeys[:]
	} else if role == suisdk.TimelockRoleBypasser {
		keys = bypasserKeys[:]
	} else {
		s.T().Fatalf("Unsupported role: %v", role)
	}
	signable, err := mcms.NewSignable(&proposal, inspectorsMap)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(keys[0]))
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(keys[1]))
	s.Require().NoError(err)

	quorumMet, err := signable.ValidateSignatures(s.T().Context())
	s.Require().NoError(err, "Error validating signatures")
	s.Require().True(quorumMet, "Quorum not met")

	// Set Root
	s.T().Logf("Preparing to the root of the proposal...")
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	suiEncoder := encoders[s.chainSelector].(*suisdk.Encoder)
	executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.mcmsPackageId, role, s.accountObj, s.registryObj, s.timelockObj)
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

	if role == suisdk.TimelockRoleProposer {
		// If proposer, some time needs to pass before the proposal can be executed sleep for delay_1_min
		s.T().Logf("Sleeping for %v before executing the proposal transfer...", delay_1_min)
		time.Sleep(delay_1_min)

		timelockExecutor, err := suisdk.NewTimelockExecutor(
			s.client,
			s.signer,
			s.mcmsPackageId,
			s.registryObj,
			s.accountObj,
		)

		s.Require().NoError(err, "creating timelock executor for Sui mcms contract")
		timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
			s.chainSelector: timelockExecutor,
		}
		timelockExecutable, err := mcms.NewTimelockExecutable(s.T().Context(), timelockProposal, timelockExecutors)
		s.Require().NoError(err)
		s.T().Logf("Executing the operation through timelock...")
		txOutput, err := timelockExecutable.Execute(s.T().Context(), 0)
		s.Require().NoError(err)
		s.T().Logf("✅ Executed proposal transfer in tx: %s", txOutput.Hash)
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
