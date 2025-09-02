//go:build e2e

package sui

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

// TimelockProposalTestSuite defines the test suite for Sui timelock proposal tests
type TimelockProposalTestSuite struct {
	SuiTestSuite
}

func (s *TimelockProposalTestSuite) Test_Sui_TimelockProposal() {
	s.SuiTestSuite.T().Run("TimelockProposal - MCMSAccount Accept Ownership through Bypass", func(t *testing.T) {
		RunAcceptOwnershipProposal(s, suisdk.TimelockRoleBypasser)
	})

	s.SuiTestSuite.T().Run("TimelockProposal - MCMSAccount Accept Ownership through Schedule", func(t *testing.T) {
		RunAcceptOwnershipProposal(s, suisdk.TimelockRoleProposer)
	})
}

func RunAcceptOwnershipProposal(s *TimelockProposalTestSuite, role suisdk.TimelockRole) {
	s.SuiTestSuite.T().Logf("Running accept ownership proposal with role: %v", role)
	s.SuiTestSuite.DeployMCMSContract()

	bypasserCount := 2
	bypasserQuorum := 2
	bypasserConfig := CreateBypasserConfig(bypasserCount, uint8(bypasserQuorum))
	proposerCount := 3
	proposerQuorum := 2
	proposerConfig := CreateProposerConfig(proposerCount, uint8(proposerQuorum))

	// Set config
	{
		configurer, err := suisdk.NewConfigurer(s.SuiTestSuite.client, s.SuiTestSuite.signer, suisdk.TimelockRoleBypasser, s.SuiTestSuite.mcmsPackageID, s.SuiTestSuite.ownerCapObj, uint64(s.SuiTestSuite.chainSelector))
		s.SuiTestSuite.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj, bypasserConfig.Config, true)
		s.SuiTestSuite.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.SuiTestSuite.client, s.SuiTestSuite.signer, suisdk.TimelockRoleProposer, s.SuiTestSuite.mcmsPackageID, s.SuiTestSuite.ownerCapObj, uint64(s.SuiTestSuite.chainSelector))
		s.SuiTestSuite.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj, proposerConfig.Config, true)
		s.SuiTestSuite.Require().NoError(err, "setting config on Sui mcms contract")
	}

	// Init transfer ownership
	{
		tx, err := s.SuiTestSuite.mcmsAccount.TransferOwnershipToSelf(
			s.SuiTestSuite.T().Context(),
			&bind.CallOpts{
				Signer:           s.SuiTestSuite.signer,
				WaitForExecution: true,
			},
			bind.Object{Id: s.SuiTestSuite.ownerCapObj},
			bind.Object{Id: s.SuiTestSuite.accountObj},
		)
		s.SuiTestSuite.Require().NoError(err, "Failed to transfer ownership to self")
		s.SuiTestSuite.Require().NotEmpty(tx, "Transaction should not be empty")
	}

	var timelockProposal *mcms.TimelockProposal

	delay_5_secs := time.Second * 5
	// Create a timelock proposal accepting the ownership transfer

	// Get the accept ownership call information and build the MCMS Operation
	encodedCall, err := s.SuiTestSuite.mcmsAccount.Encoder().AcceptOwnershipAsTimelock(bind.Object{Id: s.SuiTestSuite.accountObj})
	s.SuiTestSuite.Require().NoError(err)

	callBytes := []byte{}
	if len(encodedCall.CallArgs) > 0 && encodedCall.CallArgs[0].CallArg.Pure != nil {
		// TODO: Are we getting the right bytes here? callbytes are the entire bytes of the call
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
	s.SuiTestSuite.Require().NoError(err)
	op := types.BatchOperation{
		ChainSelector: s.SuiTestSuite.chainSelector,
		Transactions:  []types.Transaction{transaction},
	}

	inspector, err := suisdk.NewInspector(s.SuiTestSuite.client, s.SuiTestSuite.signer, s.SuiTestSuite.mcmsPackageID, role)
	s.SuiTestSuite.Require().NoError(err, "creating inspector for op count query")

	// Get the actual current operation count from the contract
	currentOpCount, err := inspector.GetOpCount(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj)
	s.SuiTestSuite.Require().NoError(err, "Failed to get current operation count")
	s.SuiTestSuite.T().Logf("🔍 CURRENT operation count: %d", currentOpCount)

	var action types.TimelockAction
	var delay *types.Duration
	switch role {
	case suisdk.TimelockRoleProposer:
		action = types.TimelockActionSchedule
		delayDuration := types.NewDuration(delay_5_secs)
		delay = &delayDuration
	case suisdk.TimelockRoleBypasser:
		action = types.TimelockActionBypass
	case suisdk.TimelockRoleCanceller:
		s.SuiTestSuite.T().Fatalf("TimelockRoleCanceller is not yet supported in this test")
	default:
		s.SuiTestSuite.T().Fatalf("Unsupported role: %v", role)
	}

	proposalConfig := ProposalBuilderConfig{
		Version:        "v1",
		Description:    "Accept ownership via timelock",
		ChainSelector:  s.SuiTestSuite.chainSelector,
		mcmsPackageID:  s.SuiTestSuite.mcmsPackageID,
		Role:           role,
		CurrentOpCount: currentOpCount,
		Action:         action,
		Delay:          delay,
	}

	acceptOwnershipProposalBuilder := CreateTimelockProposalBuilder(proposalConfig, []types.BatchOperation{op})
	timelockProposal, err = acceptOwnershipProposalBuilder.Build()
	s.SuiTestSuite.Require().NoError(err)

	// Sign the proposal, set root and execute proposal operations

	// Convert the Timelock Proposal into a MCMS Proposal
	timelockConverter, err := suisdk.NewTimelockConverter(s.SuiTestSuite.client, s.SuiTestSuite.signer, s.SuiTestSuite.mcmsPackageID)
	s.SuiTestSuite.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.SuiTestSuite.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(s.SuiTestSuite.T().Context(), convertersMap)
	s.SuiTestSuite.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.SuiTestSuite.chainSelector: inspector,
	}

	s.SuiTestSuite.T().Logf("Signing the proposal...")

	var keys []*ecdsa.PrivateKey
	var quorum int
	switch role {
	case suisdk.TimelockRoleProposer:
		keys = proposerConfig.Keys
		quorum = proposerQuorum
	case suisdk.TimelockRoleBypasser:
		keys = bypasserConfig.Keys
		quorum = bypasserQuorum
	case suisdk.TimelockRoleCanceller:
		s.SuiTestSuite.T().Fatalf("TimelockRoleCanceller is not yet supported in this test")
	default:
		s.SuiTestSuite.T().Fatalf("Unsupported role: %v", role)
	}
	signable, err := SignProposal(&proposal, inspectorsMap, keys, quorum)
	s.SuiTestSuite.Require().NoError(err)

	// Need to query inspector with MCMS state object ID
	quorumMet, err := signable.ValidateSignatures(s.SuiTestSuite.T().Context())
	s.SuiTestSuite.Require().NoError(err, "Error validating signatures")
	s.SuiTestSuite.Require().True(quorumMet, "Quorum not met")

	// Set Root
	s.SuiTestSuite.T().Logf("Preparing to the root of the proposal...")
	encoders, err := proposal.GetEncoders()
	s.SuiTestSuite.Require().NoError(err)
	suiEncoder := encoders[s.SuiTestSuite.chainSelector].(*suisdk.Encoder)
	executor, err := suisdk.NewExecutor(s.SuiTestSuite.client, s.SuiTestSuite.signer, suiEncoder, s.SuiTestSuite.mcmsPackageID, role, s.SuiTestSuite.mcmsObj, s.SuiTestSuite.accountObj, s.SuiTestSuite.registryObj, s.SuiTestSuite.timelockObj)
	s.SuiTestSuite.Require().NoError(err, "creating executor for Sui mcms contract")
	executors := map[types.ChainSelector]sdk.Executor{
		s.SuiTestSuite.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.SuiTestSuite.Require().NoError(err, "Error creating executable")

	s.SuiTestSuite.T().Logf("Setting the root of the proposal...")

	s.SuiTestSuite.T().Logf("=== DEBUG: Proposal Details ===")
	merkleTree, err := proposal.MerkleTree()
	s.SuiTestSuite.Require().NoError(err, "Failed to get merkle tree")
	s.SuiTestSuite.T().Logf("Merkle Tree Root: %x", merkleTree.Root)
	s.SuiTestSuite.T().Logf("Proposal ValidUntil: %d", proposal.ValidUntil)
	s.SuiTestSuite.T().Logf("Number of Operations: %d", len(proposal.Operations))

	for chainSel, metadata := range proposal.ChainMetadata {
		s.SuiTestSuite.T().Logf("Chain %d metadata - StartingOpCount: %d, MCMAddress: %s",
			chainSel, metadata.StartingOpCount, metadata.MCMAddress)
	}

	for i, op := range proposal.Operations {
		s.SuiTestSuite.T().Logf("Operation %d: ChainSelector=%d, To=%s, DataLen=%d",
			i, op.ChainSelector, op.Transaction.To, len(op.Transaction.Data))
	}

	signingHash, err := proposal.SigningHash()
	s.SuiTestSuite.Require().NoError(err, "Failed to get signing hash")
	s.SuiTestSuite.T().Logf("Proposal Signing Hash: %x", signingHash)

	quorumMet, err = signable.ValidateSignatures(s.SuiTestSuite.T().Context())
	s.SuiTestSuite.Require().NoError(err, "Error validating signatures")
	s.SuiTestSuite.Require().True(quorumMet, "Quorum not met")

	result, err := executable.SetRoot(s.SuiTestSuite.T().Context(), s.SuiTestSuite.chainSelector)
	s.SuiTestSuite.Require().NoError(err)

	s.SuiTestSuite.T().Logf("✅ SetRoot in tx: %s", result.Hash)

	s.SuiTestSuite.T().Logf("Executing the proposal operations...")
	// Execute
	for i := range proposal.Operations {
		s.SuiTestSuite.T().Logf("Executing operation: %v", i)
		txOutput, execErr := executable.Execute(s.SuiTestSuite.T().Context(), i)
		s.SuiTestSuite.Require().NoError(execErr)
		s.SuiTestSuite.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)
	}

	if role == suisdk.TimelockRoleProposer {
		// If proposer, some time needs to pass before the proposal can be executed sleep for delay_5_secs
		s.SuiTestSuite.T().Logf("Sleeping for %v before executing the proposal transfer...", delay_5_secs)
		time.Sleep(delay_5_secs)

		timelockExecutor, tErr := suisdk.NewTimelockExecutor(
			s.SuiTestSuite.client,
			s.SuiTestSuite.signer,
			s.SuiTestSuite.mcmsPackageID,
			s.SuiTestSuite.registryObj,
			s.SuiTestSuite.accountObj,
		)

		s.SuiTestSuite.Require().NoError(tErr, "creating timelock executor for Sui mcms contract")
		timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
			s.SuiTestSuite.chainSelector: timelockExecutor,
		}
		timelockExecutable, execErr := mcms.NewTimelockExecutable(s.SuiTestSuite.T().Context(), timelockProposal, timelockExecutors)
		s.SuiTestSuite.Require().NoError(execErr)
		s.SuiTestSuite.T().Logf("Executing the operation through timelock...")
		txOutput, terr := timelockExecutable.Execute(s.SuiTestSuite.T().Context(), 0, mcms.WithCallProxy(s.SuiTestSuite.timelockObj))
		s.SuiTestSuite.Require().NoError(terr)
		s.SuiTestSuite.T().Logf("✅ Executed proposal transfer in tx: %s", txOutput.Hash)
	}
	// Complete the proposal transfer
	s.SuiTestSuite.T().Logf("Completing the proposal transfer...")
	tx, err := s.SuiTestSuite.mcmsAccount.ExecuteOwnershipTransfer(s.SuiTestSuite.T().Context(), &bind.CallOpts{
		Signer:           s.SuiTestSuite.signer,
		WaitForExecution: true,
	}, bind.Object{Id: s.SuiTestSuite.ownerCapObj}, bind.Object{Id: s.SuiTestSuite.accountObj}, bind.Object{Id: s.SuiTestSuite.registryObj}, s.SuiTestSuite.mcmsPackageID)
	s.SuiTestSuite.Require().NoError(err, "Failed to execute ownership transfer")
	s.SuiTestSuite.Require().NotEmpty(tx, "Transaction should not be empty")
	s.SuiTestSuite.T().Logf("✅ Executed ownership transfer in tx: %s", tx.Digest)

	// Check owner
	owner, err := bind.ReadObject(s.SuiTestSuite.T().Context(), s.SuiTestSuite.accountObj, s.SuiTestSuite.client)
	s.SuiTestSuite.Require().NoError(err)
	s.SuiTestSuite.Require().Equal(s.SuiTestSuite.mcmsPackageID, owner.Data.Content.Fields["owner"], "Owner should be the mcms package ID")

	// Check op count got incremented by 1
	postOpCount, err := inspector.GetOpCount(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj)
	s.SuiTestSuite.Require().NoError(err, "Failed to get post operation count")
	s.SuiTestSuite.T().Logf("🔍 POST operation count: %d", postOpCount)
	s.SuiTestSuite.Require().Equal(currentOpCount+1, postOpCount, "Operation count should be incremented by 1")
}
