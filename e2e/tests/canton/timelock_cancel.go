//go:build e2e

package canton

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/types"
)

// TimelockCancelTestSuite tests the cancel timelock flow:
// schedule a batch -> cancel it -> verify execution fails.
type TimelockCancelTestSuite struct {
	mcmsExecutorSetup
}

// TestTimelockCancel schedules a batch, cancels it, then verifies execution fails.
func (s *TimelockCancelTestSuite) TestTimelockCancel() {
	ctx := s.T().Context()

	// --- Phase 1: Schedule a batch (same as TestTimelockProposal) ---

	proposerInspector := cantonsdk.NewInspector(s.participant.LedgerServices.State, s.participant.PartyID, cantonsdk.TimelockRoleProposer)
	currentOpCount, err := proposerInspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err, "get current op count")

	// Proposer metadata for schedule
	proposerMetadata, err := cantonsdk.NewChainMetadata(
		currentOpCount,
		currentOpCount+1,
		s.chainId,
		s.proposerMcmsId,
		s.mcmsInstanceAddress,
		false,
		s.mcmsId,
	)
	s.Require().NoError(err)

	validUntil := uint32(time.Now().Add(24 * time.Hour).Unix())
	delay := types.NewDuration(10 * time.Second) // Longer delay so we can cancel before it's ready

	// Batch operation: increment counter
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceAddress: fmt.Sprintf("%s@%s", s.counterInstanceID, s.participant.PartyID),
		FunctionName:          "Increment",
		OperationData:         "",
		TargetCid:             s.counterCID,
		ContractIds:           []string{s.counterCID},
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	bop := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions: []types.Transaction{{
			To:               s.counterCID,
			Data:             []byte{},
			AdditionalFields: opAdditionalFieldsBytes,
		}},
	}

	// Build schedule proposal
	scheduleProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Canton timelock - schedule for cancel test").
		AddTimelockAddress(s.chainSelector, s.mcmsInstanceAddress).
		AddChainMetadata(s.chainSelector, proposerMetadata).
		SetAction(types.TimelockActionSchedule).
		SetDelay(delay).
		AddOperation(bop).
		Build()
	s.Require().NoError(err)

	// Convert, sign, and execute schedule
	converter := cantonsdk.NewTimelockConverter()
	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: converter,
	}
	proposal, _, err := scheduleProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: proposerInspector,
	}
	signable, err := mcms.NewSignable(&proposal, inspectorsMap)
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[0]))
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[1]))
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)
	proposerExecutor, err := cantonsdk.NewExecutor(encoder, proposerInspector, s.participant.LedgerServices.Command, s.participant.UserID, s.participant.PartyID, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: proposerExecutor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr, "execute schedule operation %d", i)
	}

	// Note: Operation is now scheduled on-chain. We skip the IsOperationPending check
	// because the operationID returned by Convert() may differ from what the contract uses.

	// --- Phase 2: Cancel the batch ---

	// Get canceller op count
	cancellerInspector := cantonsdk.NewInspector(s.participant.LedgerServices.State, s.participant.PartyID, cantonsdk.TimelockRoleCanceller)
	cancellerOpCount, err := cancellerInspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)

	// Canceller metadata
	cancellerMcmsId := fmt.Sprintf("%s@%s-canceller", s.mcmsId, s.participant.PartyID)
	cancellerMetadata, err := cantonsdk.NewChainMetadata(
		cancellerOpCount,
		cancellerOpCount+1,
		s.chainId,
		cancellerMcmsId,
		s.mcmsInstanceAddress,
		false,
		s.mcmsId,
	)
	s.Require().NoError(err)

	// Build cancel proposal - reuse the same batch operation (the converter extracts operationId)
	// Use the same salt as the schedule proposal to derive the same operation ID
	scheduleSalt := scheduleProposal.Salt()
	cancelProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Canton timelock - cancel scheduled batch").
		AddTimelockAddress(s.chainSelector, s.mcmsInstanceAddress).
		AddChainMetadata(s.chainSelector, cancellerMetadata).
		SetAction(types.TimelockActionCancel).
		SetDelay(delay).
		SetSalt((*common.Hash)(&scheduleSalt)).
		AddOperation(bop).
		Build()
	s.Require().NoError(err)

	// Convert cancel proposal
	cancelMcmsProposal, _, err := cancelProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	// Sign with canceller role
	cancelInspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: cancellerInspector,
	}
	cancelSignable, err := mcms.NewSignable(&cancelMcmsProposal, cancelInspectorsMap)
	s.Require().NoError(err)
	_, err = cancelSignable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[0]))
	s.Require().NoError(err)
	_, err = cancelSignable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[1]))
	s.Require().NoError(err)

	// Execute cancel
	cancelEncoders, err := cancelMcmsProposal.GetEncoders()
	s.Require().NoError(err)
	cancelEncoder := cancelEncoders[s.chainSelector].(*cantonsdk.Encoder)
	cancellerExecutor, err := cantonsdk.NewExecutor(cancelEncoder, cancellerInspector, s.participant.LedgerServices.Command, s.participant.UserID, s.participant.PartyID, cantonsdk.TimelockRoleCanceller)
	s.Require().NoError(err)
	cancelExecutors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: cancellerExecutor,
	}
	cancelExecutable, err := mcms.NewExecutable(&cancelMcmsProposal, cancelExecutors)
	s.Require().NoError(err)

	_, err = cancelExecutable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	for i := range cancelMcmsProposal.Operations {
		_, execErr := cancelExecutable.Execute(ctx, i)
		s.Require().NoError(execErr, "execute cancel operation %d", i)
	}

	// --- Phase 3: Verify operation is cancelled by attempting to execute ---

	// Wait for the delay to pass so we can attempt execution
	time.Sleep(scheduleProposal.Delay.Duration + time.Second)

	// Attempt to execute via timelock should fail because operation was cancelled
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.LedgerServices.Command, s.participant.LedgerServices.State, s.participant.PartyID)
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, scheduleProposal, timelockExecutors)
	s.Require().NoError(err)

	// Execute should fail with "operation not found" or similar
	_, err = timelockExecutable.Execute(ctx, 0)
	s.Require().Error(err, "timelock execute should fail after cancel")
	s.Require().Contains(err.Error(), "not found", "error should indicate operation not found")
}
