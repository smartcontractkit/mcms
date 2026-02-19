//go:build e2e

package canton

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/types"
)

// TimelockProposalTestSuite defines the test suite for Canton timelock proposal flow:
// build proposal -> Convert -> sign -> SetRoot -> Execute (schedule) -> TimelockExecutable.Execute.
// Embeds mcmsExecutorSetup (not MCMSExecutorTestSuite) so only TestTimelockProposal runs when the suite runs.
type TimelockProposalTestSuite struct {
	mcmsExecutorSetup
}

// TestTimelockProposal runs the full timelock flow: build a Schedule proposal (increment counter),
// convert to MCMS proposal, sign, set root, execute (schedule batch), then execute via timelock.
// Fails at Convert() until Canton TimelockConverter is implemented (Phase C).
func (s *TimelockProposalTestSuite) TestTimelockProposal() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleProposer)
	currentOpCount, err := inspector.GetOpCount(ctx, s.mcmsContractID)
	s.Require().NoError(err, "get current op count")

	// Canton chain metadata for timelock (Proposer schedule)
	metadata, err := cantonsdk.NewChainMetadata(
		currentOpCount,
		currentOpCount+1,
		s.chainId,
		s.proposerMcmsId,
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	validUntil := uint32(time.Now().Add(24 * time.Hour).Unix())
	delay := types.NewDuration(2 * time.Second)

	// Batch operation: increment counter (same shape as in TestSetRootAndExecuteCounterOp)
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: fmt.Sprintf("%s@%s", s.counterInstanceID, s.participant.Party),
		FunctionName:     "increment",
		OperationData:    "",
		TargetCid:        s.counterCID,
		ContractIds:      []string{s.counterCID},
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

	// Build timelock proposal (Schedule action)
	timelockProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Canton timelock - schedule counter increment").
		AddTimelockAddress(s.chainSelector, s.mcmsContractID).
		AddChainMetadata(s.chainSelector, metadata).
		SetAction(types.TimelockActionSchedule).
		SetDelay(delay).
		AddOperation(bop).
		Build()
	s.Require().NoError(err)

	// Convert timelock proposal to MCMS proposal (requires Canton TimelockConverter implementation)
	converter := cantonsdk.NewTimelockConverter()
	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: converter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err, "Convert: Canton TimelockConverter must be implemented (Phase C)")

	// Sign proposal
	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}
	signable, err := mcms.NewSignable(&proposal, inspectorsMap)
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[0]))
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[1]))
	s.Require().NoError(err)
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet, "quorum not met")

	// Set root
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)
	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)
	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	// SetRoot archives the old MCMS and creates a new one; extract new contract ID and update proposal/state
	rawData, ok := txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok, "SetRoot must return NewMCMSContractID")
	s.Require().NotEmpty(newMCMSContractID)
	oldMCMSContractID := s.mcmsContractID
	s.mcmsContractID = newMCMSContractID

	// Update proposal so Execute uses the new MCMS contract ID (executable holds a pointer to proposal)
	meta := proposal.ChainMetadata[s.chainSelector]
	meta.MCMAddress = newMCMSContractID
	proposal.ChainMetadata[s.chainSelector] = meta
	for i := range proposal.Operations {
		op := &proposal.Operations[i]
		op.Transaction.To = newMCMSContractID
		var af cantonsdk.AdditionalFields
		if err := json.Unmarshal(op.Transaction.AdditionalFields, &af); err == nil {
			af.TargetCid = newMCMSContractID
			if len(af.ContractIds) > 0 && af.ContractIds[0] == oldMCMSContractID {
				af.ContractIds[0] = newMCMSContractID
			}
			op.Transaction.AdditionalFields, _ = json.Marshal(af)
		}
	}

	// Execute proposal operations (schedules the batch on-chain)
	for i := range proposal.Operations {
		txExecute, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr, "execute scheduled operation %d", i)
		// ExecuteOp may recreate MCMS; keep suite state in sync for GetOpCount later
		if rd, ok := txExecute.RawData.(map[string]any); ok {
			if cid, ok := rd["NewMCMSContractID"].(string); ok && cid != "" {
				s.mcmsContractID = cid
			}
		}
	}

	// Timelock execution: wait for ready then execute batch
	mcmsPkgID := ""
	if len(s.packageIDs) > 0 {
		mcmsPkgID = s.packageIDs[0]
	}
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.CommandServiceClient, s.participant.Party, mcmsPkgID)
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	// Wait until operation is ready (delay has passed)
	s.Require().NoError(timelockExecutable.IsReady(ctx), "timelock operation should become ready")

	// Execute the scheduled batch via timelock
	for i := range timelockProposal.Operations {
		_, terr := timelockExecutable.Execute(ctx, i)
		s.Require().NoError(terr, "timelock execute operation %d", i)
	}

	// Verify: op count increased (and counter was incremented when inspection is available)
	postOpCount, err := inspector.GetOpCount(ctx, s.mcmsContractID)
	s.Require().NoError(err)
	s.Require().Equal(currentOpCount+1, postOpCount, "op count should increment after timelock execute")
}
