//go:build e2e

package sui

import (
	"time"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

// TimelockProposalTestSuite defines the test suite for Sui timelock proposal tests
type TimelockCancelProposalTestSuite struct {
	SuiTestSuite
}

func (s *TimelockCancelProposalTestSuite) Test_Sui_TimelockCancelProposal() {
	s.DeployMCMSContract()

	proposerCount := 3
	proposerQuorum := 2
	proposerConfig := CreateConfig(proposerCount, uint8(proposerQuorum))

	// Set config
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

	// Get the accept ownership call information and build the MCMS Operation
	encodedCall, err := s.mcmsAccount.Encoder().AcceptOwnershipAsTimelock(bind.Object{Id: s.accountObj})
	s.Require().NoError(err)

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
	s.Require().NoError(err)
	op := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{transaction},
	}

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "creating inspector for op count query")

	// Get the actual current operation count from the contract
	currentOpCount, err := inspector.GetOpCount(s.T().Context(), s.mcmsObj)
	s.Require().NoError(err, "Failed to get current operation count")

	action := types.TimelockActionSchedule
	delay5s := time.Second * 5
	delay := types.NewDuration(delay5s)

	proposalConfig := ProposalBuilderConfig{
		Version:        "v1",
		Description:    "Accept ownership via timelock",
		ChainSelector:  s.chainSelector,
		McmsObjID:      s.mcmsObj,
		TimelockObjID:  s.timelockObj,
		AccountObjID:   s.accountObj,
		RegistryObjID:  s.registryObj,
		McmsPackageID:  s.mcmsPackageID,
		Role:           suisdk.TimelockRoleProposer,
		CurrentOpCount: currentOpCount,
		Action:         action,
		Delay:          &delay,
	}

	acceptOwnershipProposalBuilder := CreateTimelockProposalBuilder(s.T(), proposalConfig, []types.BatchOperation{op})
	timelockProposal, err := acceptOwnershipProposalBuilder.Build()
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

	keys := proposerConfig.Keys
	quorum := proposerQuorum
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
	executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.entrypointArgEncoder, s.mcmsPackageID, suisdk.TimelockRoleProposer, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
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
		_, eErr := executable.Execute(s.T().Context(), i)
		s.Require().NoError(eErr, "Error executing operation")

		// Create timelock inspector to check operation status
		timelockInspector, eErr := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
		s.Require().NoError(eErr, "Failed to create timelock inspector")

		timelockExecutor, eErr := suisdk.NewTimelockExecutor(
			s.client,
			s.signer,
			s.entrypointArgEncoder,
			s.mcmsPackageID,
			s.registryObj,
			s.accountObj,
		)
		s.Require().NoError(eErr, "creating timelock executor for Sui mcms contract")
		timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
			s.chainSelector: timelockExecutor,
		}
		timelockExecutable, eErr := mcms.NewTimelockExecutable(s.T().Context(), timelockProposal, timelockExecutors)
		s.Require().NoError(eErr)

		// Get the operation ID that was scheduled
		scheduledOpID, eErr := timelockExecutable.GetOpID(s.T().Context(), 0, op, s.chainSelector)
		s.Require().NoError(eErr, "Failed to get operation ID")

		// The operation should still exist
		exists, eErr := timelockInspector.IsOperation(s.T().Context(), s.timelockObj, scheduledOpID)
		s.Require().NoError(eErr, "IsOperation should not return an error")
		s.Require().True(exists, "Operation should still exist after delay")

		// The operation should still be pending (scheduled but not executed)
		isPending, eErr := timelockInspector.IsOperationPending(s.T().Context(), s.timelockObj, scheduledOpID)
		s.Require().NoError(eErr, "IsOperationPending should not return an error")
		s.Require().True(isPending, "Operation should still be pending after delay")

		delay7s := time.Second * 7
		time.Sleep(delay7s)

		// The operation should NOW be ready (delay has passed)
		isReady, eErr := timelockInspector.IsOperationReady(s.T().Context(), s.timelockObj, scheduledOpID)
		s.Require().NoError(eErr, "IsOperationReady should not return an error")
		s.Require().True(isReady, "Operation should be ready after delay passes")

		// The operation should still NOT be done (not executed yet)
		isDone, eErr := timelockInspector.IsOperationDone(s.T().Context(), s.timelockObj, scheduledOpID)
		s.Require().NoError(eErr, "IsOperationDone should not return an error")
		s.Require().False(isDone, "Operation should not be done before execution")
	}

	// Cancel the proposal
	cancellerCount := 3
	cancellerQuorum := 2
	cancellerConfig := CreateConfig(cancellerCount, uint8(cancellerQuorum))

	// Set config
	{
		configurer, cErr := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleCanceller, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(cErr, "creating configurer for Sui mcms contract")
		_, cErr = configurer.SetConfig(s.T().Context(), s.mcmsObj, cancellerConfig.Config, true)
		s.Require().NoError(cErr, "setting config on Sui mcms contract")
	}

	// Get the current operation count for the cancellation proposal
	cancelOpCountInspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, suisdk.TimelockRoleCanceller)
	s.Require().NoError(err, "creating canceller inspector for op count")

	currentCancelOpCount, err := cancelOpCountInspector.GetOpCount(s.T().Context(), s.mcmsObj)
	s.Require().NoError(err, "Failed to get current operation count for cancellation")

	metadata, err := suisdk.NewChainMetadata(currentCancelOpCount, suisdk.TimelockRoleCanceller, s.mcmsPackageID, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err, "Failed to create chain metadata for cancellation proposal")

	cancelTimelockProposal, err := timelockProposal.DeriveCancellationProposal(map[types.ChainSelector]types.ChainMetadata{
		s.chainSelector: metadata,
	})
	s.Require().NoError(err, "Failed to derive cancellation proposal")

	cancelProposal, opId, err := cancelTimelockProposal.Convert(s.T().Context(), convertersMap)
	s.Require().NoError(err, "Failed to convert cancellation proposal")

	// Sign the cancellation proposal
	cancelInspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, suisdk.TimelockRoleCanceller)
	s.Require().NoError(err, "creating canceller inspector")

	cancelInspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: cancelInspector,
	}

	cancelSignable, err := SignProposal(&cancelProposal, cancelInspectorsMap, cancellerConfig.Keys, cancellerQuorum)
	s.Require().NoError(err, "Failed to sign cancellation proposal")

	// Validate cancellation proposal signatures
	quorumMet, err = cancelSignable.ValidateSignatures(s.T().Context())
	s.Require().NoError(err, "Error validating cancellation signatures")
	s.Require().True(quorumMet, "Cancellation quorum not met")

	// Set root for cancellation proposal
	cancelEncoders, err := cancelProposal.GetEncoders()
	s.Require().NoError(err)
	cancelSuiEncoder := cancelEncoders[s.chainSelector].(*suisdk.Encoder)
	cancelExecutor, err := suisdk.NewExecutor(s.client, s.signer, cancelSuiEncoder, s.entrypointArgEncoder, s.mcmsPackageID, suisdk.TimelockRoleCanceller, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err, "creating canceller executor for Sui mcms contract")

	cancelExecutors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: cancelExecutor,
	}
	cancelExecutable, err := mcms.NewExecutable(&cancelProposal, cancelExecutors)
	s.Require().NoError(err, "Error creating cancellation executable")

	_, err = cancelExecutable.SetRoot(s.T().Context(), s.chainSelector)
	s.Require().NoError(err, "Failed to set root for cancellation proposal")

	// Execute the cancellation proposal
	for i := range cancelProposal.Operations {
		_, execErr := cancelExecutable.Execute(s.T().Context(), i)
		s.Require().NoError(execErr, "Error executing cancellation operation")
	}

	// Verify that the original operation is canceled
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

	// The operation should NOT be pending anymore (canceled)
	isPending, err := timelockInspector.IsOperationPending(s.T().Context(), s.timelockObj, scheduledOpID)
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().False(isPending, "Operation should not be pending after cancellation")

	// The operation should NOT be ready (canceled)
	isReady, err := timelockInspector.IsOperationReady(s.T().Context(), s.timelockObj, scheduledOpID)
	s.Require().NoError(err, "IsOperationReady should not return an error")
	s.Require().False(isReady, "Operation should not be ready after cancellation")

	// The operation should NOT be done (canceled, not executed)
	isDone, err := timelockInspector.IsOperationDone(s.T().Context(), s.timelockObj, scheduledOpID)
	s.Require().NoError(err, "IsOperationDone should not return an error")
	s.Require().False(isDone, "Operation should not be done after cancellation")

	// Verify that trying to execute the original operation now fails
	_, execErr = timelockExecutable.Execute(s.T().Context(), 0, mcms.WithCallProxy(s.timelockObj))
	s.Require().Error(execErr, "Executing canceled operation should fail")

	s.T().Logf("✅ Successfully canceled timelock operation with ID: %s", opId)
}
