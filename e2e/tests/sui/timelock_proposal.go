//go:build e2e

package sui

import (
	"crypto/ecdsa"
	"time"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

// TimelockProposalTestSuite defines the test suite for Sui timelock proposal tests
type TimelockProposalTestSuite struct {
	TestSuite
}

func (s *TimelockProposalTestSuite) TestTimelockProposal() {
	s.Run("TimelockProposal - MCMSAccount Accept Ownership through Bypass", func() {
		RunAcceptOwnershipProposal(s, suisdk.TimelockRoleBypasser)
	})

	s.Run("TimelockProposal - MCMSAccount Accept Ownership through Schedule", func() {
		RunAcceptOwnershipProposal(s, suisdk.TimelockRoleProposer)
	})
}

func RunAcceptOwnershipProposal(s *TimelockProposalTestSuite, role suisdk.TimelockRole) {
	s.DeployMCMSContract()

	bypasserCount := 2
	bypasserQuorum := 2
	bypasserConfig := CreateConfig(bypasserCount, uint8(bypasserQuorum))
	proposerCount := 3
	proposerQuorum := 2
	proposerConfig := CreateConfig(proposerCount, uint8(proposerQuorum))

	// Set config
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleBypasser, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcmsObj, bypasserConfig.Config, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcmsObj, proposerConfig.Config, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}

	// Init transfer ownership
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

	delay5s := time.Second * 5
	// Create a timelock proposal accepting the ownership transfer

	// Get the accept ownership call information and build the MCMS Operation
	encodedCall, err := s.mcmsAccount.Encoder().AcceptOwnershipAsTimelock(bind.Object{Id: s.accountObj})
	s.Require().NoError(err)

	callBytes := s.extractByteArgsFromEncodedCall(*encodedCall)

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

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, role)
	s.Require().NoError(err, "creating inspector for op count query")

	// Get the actual current operation count from the contract
	currentOpCount, err := inspector.GetOpCount(s.T().Context(), s.mcmsObj)
	s.Require().NoError(err, "Failed to get current operation count")

	var action types.TimelockAction
	var delay *types.Duration
	switch role {
	case suisdk.TimelockRoleProposer:
		action = types.TimelockActionSchedule
		delayDuration := types.NewDuration(delay5s)
		delay = &delayDuration
	case suisdk.TimelockRoleBypasser:
		action = types.TimelockActionBypass
	case suisdk.TimelockRoleCanceller:
		s.T().Fatalf("TimelockRoleCanceller is not yet supported in this test")
	default:
		s.T().Fatalf("Unsupported role: %v", role)
	}

	proposalConfig := ProposalBuilderConfig{
		Version:            "v1",
		Description:        "Accept ownership via timelock",
		ChainSelector:      s.chainSelector,
		McmsObjID:          s.mcmsObj,
		TimelockObjID:      s.timelockObj,
		McmsPackageID:      s.mcmsPackageID,
		AccountObjID:       s.accountObj,
		RegistryObjID:      s.registryObj,
		DeployerStateObjID: s.depStateObj,
		Role:               role,
		CurrentOpCount:     currentOpCount,
		Action:             action,
		Delay:              delay,
	}

	acceptOwnershipProposalBuilder := CreateTimelockProposalBuilder(s.T(), proposalConfig, []types.BatchOperation{op})
	timelockProposal, err = acceptOwnershipProposalBuilder.Build()
	s.Require().NoError(err)

	// Sign the proposal, set root and execute proposal operations

	// Convert the Timelock Proposal into a MCMS Proposal
	timelockConverter, err := suisdk.NewTimelockConverter()
	s.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(s.T().Context(), convertersMap)
	s.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}

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
		s.T().Fatalf("TimelockRoleCanceller is not yet supported in this test")
	default:
		s.T().Fatalf("Unsupported role: %v", role)
	}
	signable, err := SignProposal(&proposal, inspectorsMap, keys, quorum)
	s.Require().NoError(err)

	// Need to query inspector with MCMS state object ID
	quorumMet, err := signable.ValidateSignatures(s.T().Context())
	s.Require().NoError(err, "Error validating signatures")
	s.Require().True(quorumMet, "Quorum not met")

	// Set Root
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	suiEncoder := encoders[s.chainSelector].(*suisdk.Encoder)
	executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.entrypointArgEncoder, s.mcmsPackageID, role, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err, "creating executor for Sui mcms contract")
	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err, "Error creating executable")

	quorumMet, err = signable.ValidateSignatures(s.T().Context())
	s.Require().NoError(err, "Error validating signatures")
	s.Require().True(quorumMet, "Quorum not met")

	_, err = executable.SetRoot(s.T().Context(), s.chainSelector)
	s.Require().NoError(err)

	// Execute
	for i := range proposal.Operations {
		_, execErr := executable.Execute(s.T().Context(), i)
		s.Require().NoError(execErr, "Error executing operation")

		if role == suisdk.TimelockRoleProposer {
			// If proposer, some time needs to pass before the proposal can be executed sleep for delay5s

			// Create timelock inspector to check operation status
			timelockInspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
			s.Require().NoError(err, "Failed to create timelock inspector")

			timelockExecutor, tErr := suisdk.NewTimelockExecutor(
				s.client,
				s.signer,
				s.entrypointArgEncoder,
				s.mcmsPackageID,
				s.registryObj,
				s.accountObj,
			)
			s.Require().NoError(tErr, "creating timelock executor for Sui mcms contract")
			timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
				s.chainSelector: timelockExecutor,
			}
			timelockExecutable, execErr := mcms.NewTimelockExecutable(s.T().Context(), timelockProposal, timelockExecutors)
			s.Require().NoError(execErr)

			// Get the operation ID that was scheduled
			scheduledOpID, err := timelockExecutable.GetOpID(s.T().Context(), 0, op, s.chainSelector)
			s.Require().NoError(err, "Failed to get operation ID")

			// The operation should exist (be scheduled)
			exists, err := timelockInspector.IsOperation(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperation should not return an error")
			s.Require().True(exists, "Operation should exist after scheduling")

			// The operation should be pending (scheduled but not ready)
			isPending, err := timelockInspector.IsOperationPending(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationPending should not return an error")
			s.Require().True(isPending, "Operation should be pending before delay passes")

			// The operation should NOT be ready yet (delay hasn't passed)
			isReady, err := timelockInspector.IsOperationReady(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationReady should not return an error")
			s.Require().False(isReady, "Operation should not be ready before delay passes")

			// The operation should NOT be done yet
			isDone, err := timelockInspector.IsOperationDone(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationDone should not return an error")
			s.Require().False(isDone, "Operation should not be done before execution")

			time.Sleep(delay5s)

			// The operation should still exist
			exists, err = timelockInspector.IsOperation(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperation should not return an error")
			s.Require().True(exists, "Operation should still exist after delay")

			// The operation should still be pending (scheduled but not executed)
			isPending, err = timelockInspector.IsOperationPending(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationPending should not return an error")
			s.Require().True(isPending, "Operation should still be pending after delay")

			// The operation should NOW be ready (delay has passed)
			isReady, err = timelockInspector.IsOperationReady(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationReady should not return an error")
			s.Require().True(isReady, "Operation should be ready after delay passes")

			// The operation should still NOT be done (not executed yet)
			isDone, err = timelockInspector.IsOperationDone(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationDone should not return an error")
			s.Require().False(isDone, "Operation should not be done before execution")

			// Execute the operation
			_, terr := timelockExecutable.Execute(s.T().Context(), 0, mcms.WithCallProxy(s.timelockObj))
			s.Require().NoError(terr)

			// The operation should still exist
			exists, err = timelockInspector.IsOperation(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperation should not return an error")
			s.Require().True(exists, "Operation should still exist after execution")

			// The operation should NOT be pending anymore (executed)
			isPending, err = timelockInspector.IsOperationPending(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationPending should not return an error")
			s.Require().False(isPending, "Operation should not be pending after execution")

			// The operation should NOW be done (executed)
			isDone, err = timelockInspector.IsOperationDone(s.T().Context(), s.timelockObj, scheduledOpID)
			s.Require().NoError(err, "IsOperationDone should not return an error")
			s.Require().True(isDone, "Operation should be done after execution")
		}
		// Complete the proposal transfer
		tx, err := s.mcmsAccount.ExecuteOwnershipTransfer(s.T().Context(), &bind.CallOpts{
			Signer:           s.signer,
			WaitForExecution: true,
		}, bind.Object{Id: s.ownerCapObj}, bind.Object{Id: s.accountObj}, bind.Object{Id: s.registryObj}, s.mcmsPackageID)
		s.Require().NoError(err, "Failed to execute ownership transfer")
		s.Require().NotEmpty(tx, "Transaction should not be empty")

		// Check owner
		owner, err := bind.ReadObject(s.T().Context(), s.accountObj, s.client)
		s.Require().NoError(err)
		s.Require().Equal(s.mcmsPackageID, owner.Data.Content.Fields["owner"], "Owner should be the mcms package ID")

		// Check op count got incremented by 1
		postOpCount, err := inspector.GetOpCount(s.T().Context(), s.mcmsObj)
		s.Require().NoError(err, "Failed to get post operation count")
		s.Require().Equal(currentOpCount+1, postOpCount, "Operation count should be incremented by 1")
	}
}
