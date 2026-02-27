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
	currentOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err, "get current op count")

	// Canton chain metadata: multisigId = makeMcmsId(instanceId, Proposer); baseInstanceId for converter TargetInstanceId
	metadata, err := cantonsdk.NewChainMetadata(
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
	delay := types.NewDuration(2 * time.Second)

	// Batch operation: increment counter (same shape as in TestSetRootAndExecuteCounterOp)
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: fmt.Sprintf("%s@%s", s.counterInstanceID, s.participant.Party),
		FunctionName:     "Increment",
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

	// Build timelock proposal (Schedule action); timelock address is InstanceAddress hex
	timelockProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Canton timelock - schedule counter increment").
		AddTimelockAddress(s.chainSelector, s.mcmsInstanceAddress).
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
	s.Require().NotEmpty(txSetRoot.Hash)
	// No proposal mutation: proposal keeps InstanceAddress hex; executor resolves at submit time

	// Execute proposal operations (schedules the batch on-chain)
	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr, "execute scheduled operation %d", i)
	}

	// Timelock execution: wait for ready then execute batch
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.CommandServiceClient, s.participant.StateServiceClient, s.participant.Party)
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	// Wait until operation is ready (delay has passed)
	time.Sleep(timelockProposal.Delay.Duration + time.Second)
	s.Require().NoError(timelockExecutable.IsReady(ctx), "timelock operation should become ready")

	// Execute the scheduled batch via timelock
	for i := range timelockProposal.Operations {
		_, terr := timelockExecutable.Execute(ctx, i)
		s.Require().NoError(terr, "timelock execute operation %d", i)
	}

	// Verify: op count increased (inspector resolves InstanceAddress when querying)
	postOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)
	s.Require().Equal(currentOpCount+1, postOpCount, "op count should increment after timelock execute")
}
