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

// SetRootExecuteTestSuite tests the core MCMS proposal path via the Proposer role:
// build timelock proposal -> convert (ScheduleBatch) -> sign -> SetRoot -> Execute (schedule)
// -> wait for delay -> TimelockExecutable.Execute -> verify op count.
type SetRootExecuteTestSuite struct {
	mcmsExecutorSetup
}

// TestSetRootAndExecute builds a Proposer schedule proposal that increments the counter,
// sets the root, executes (schedules the batch), waits for delay, then executes via timelock.
func (s *SetRootExecuteTestSuite) TestSetRootAndExecute() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewInspector(s.participant.LedgerServices.State, []string{s.participant.PartyID}, cantonsdk.TimelockRoleProposer)
	currentOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err, "get current op count")

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
	delay := types.NewDuration(1 * time.Second)

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

	timelockProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Canton set-root-execute - schedule counter increment").
		AddTimelockAddress(s.chainSelector, s.mcmsInstanceAddress).
		AddChainMetadata(s.chainSelector, metadata).
		SetAction(types.TimelockActionSchedule).
		SetDelay(delay).
		AddOperation(bop).
		Build()
	s.Require().NoError(err)

	converter := cantonsdk.NewTimelockConverter()
	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: converter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

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

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)
	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.LedgerServices.Command, s.submittingParty, []string{s.participant.PartyID}, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txSetRoot.Hash)

	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr, "execute scheduled operation %d", i)
	}

	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.LedgerServices.Command, s.participant.LedgerServices.State, s.submittingParty, []string{s.participant.PartyID})
	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	time.Sleep(timelockProposal.Delay.Duration + time.Second)
	s.Require().NoError(timelockExecutable.IsReady(ctx), "timelock operation should become ready")

	for i := range timelockProposal.Operations {
		_, terr := timelockExecutable.Execute(ctx, i)
		s.Require().NoError(terr, "timelock execute operation %d", i)
	}

	postOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)
	s.Require().Equal(currentOpCount+1, postOpCount, "op count should increment after execute")
}

// TestSetRootAndExecuteMultipleOps builds two sequential schedule proposals to verify
// nonce/opCount handling works correctly across multiple proposal executions.
// Uses self-dispatch (UpdateMinDelay) to avoid external contract CID staleness across iterations.
func (s *SetRootExecuteTestSuite) TestSetRootAndExecuteMultipleOps() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewInspector(s.participant.LedgerServices.State, []string{s.participant.PartyID}, cantonsdk.TimelockRoleProposer)
	startOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err, "get starting op count")

	mcmsTargetInstanceAddr := fmt.Sprintf("%s@%s", s.mcmsId, s.participant.PartyID)

	for i := range uint64(2) {
		currentOpCount := startOpCount + i

		metadata, metaErr := cantonsdk.NewChainMetadata(
			currentOpCount,
			currentOpCount+1,
			s.chainId,
			s.proposerMcmsId,
			s.mcmsInstanceAddress,
			i > 0,
			s.mcmsId,
		)
		s.Require().NoError(metaErr)

		validUntil := uint32(time.Now().Add(24 * time.Hour).Unix())
		delay := types.NewDuration(1 * time.Second)

		opAdditionalFields := cantonsdk.AdditionalFields{
			TargetInstanceAddress: mcmsTargetInstanceAddr,
			FunctionName:          "UpdateMinDelay",
			OperationData:         encodeMinDelay(1),
			TargetCid:             "",
			ContractIds:           []string{},
		}
		opAdditionalFieldsBytes, marshalErr := json.Marshal(opAdditionalFields)
		s.Require().NoError(marshalErr)

		bop := types.BatchOperation{
			ChainSelector: s.chainSelector,
			Transactions: []types.Transaction{{
				To:               mcmsTargetInstanceAddr,
				Data:             []byte{},
				AdditionalFields: opAdditionalFieldsBytes,
			}},
		}

		timelockProposal, buildErr := mcms.NewTimelockProposalBuilder().
			SetVersion("v1").
			SetValidUntil(validUntil).
			SetDescription(fmt.Sprintf("Canton multi-op proposal %d - UpdateMinDelay", i+1)).
			AddTimelockAddress(s.chainSelector, s.mcmsInstanceAddress).
			AddChainMetadata(s.chainSelector, metadata).
			SetAction(types.TimelockActionSchedule).
			SetDelay(delay).
			AddOperation(bop).
			Build()
		s.Require().NoError(buildErr)

		converter := cantonsdk.NewTimelockConverter()
		convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
			s.chainSelector: converter,
		}
		proposal, _, convertErr := timelockProposal.Convert(ctx, convertersMap)
		s.Require().NoError(convertErr)

		inspectorsMap := map[types.ChainSelector]sdk.Inspector{
			s.chainSelector: inspector,
		}
		signable, signableErr := mcms.NewSignable(&proposal, inspectorsMap)
		s.Require().NoError(signableErr)
		_, signErr := signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[0]))
		s.Require().NoError(signErr)
		_, signErr = signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[1]))
		s.Require().NoError(signErr)

		encoders, encErr := proposal.GetEncoders()
		s.Require().NoError(encErr)
		encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)
		executor, execErr := cantonsdk.NewExecutor(encoder, inspector, s.participant.LedgerServices.Command, s.submittingParty, []string{s.participant.PartyID}, cantonsdk.TimelockRoleProposer)
		s.Require().NoError(execErr)
		executors := map[types.ChainSelector]sdk.Executor{
			s.chainSelector: executor,
		}
		executable, exeErr := mcms.NewExecutable(&proposal, executors)
		s.Require().NoError(exeErr)

		_, err = executable.SetRoot(ctx, s.chainSelector)
		s.Require().NoError(err)

		for j := range proposal.Operations {
			_, execErr := executable.Execute(ctx, j)
			s.Require().NoError(execErr, "execute scheduled operation %d of proposal %d", j, i+1)
		}

		timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.LedgerServices.Command, s.participant.LedgerServices.State, s.submittingParty, []string{s.participant.PartyID})
		timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
			s.chainSelector: timelockExecutor,
		}
		timelockExecutable, tlExeErr := mcms.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
		s.Require().NoError(tlExeErr)

		time.Sleep(timelockProposal.Delay.Duration + time.Second)
		s.Require().NoError(timelockExecutable.IsReady(ctx), "timelock operation %d should become ready", i+1)

		for j := range timelockProposal.Operations {
			_, terr := timelockExecutable.Execute(ctx, j)
			s.Require().NoError(terr, "timelock execute operation %d of proposal %d", j, i+1)
		}

		postOpCount, postCountErr := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
		s.Require().NoError(postCountErr)
		s.Require().Equal(currentOpCount+1, postOpCount, "op count should be %d after proposal %d", currentOpCount+1, i+1)
	}

	finalOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)
	s.Require().Equal(startOpCount+2, finalOpCount, "op count should increase by 2 after two proposals")
}

// TestSetRootInvalidSignature verifies that SetRoot fails when given an invalid signature
// (signed by a key not in the MCMS config).
func (s *SetRootExecuteTestSuite) TestSetRootInvalidSignature() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewInspector(s.participant.LedgerServices.State, []string{s.participant.PartyID}, cantonsdk.TimelockRoleProposer)
	currentOpCount, err := inspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)

	metadata, err := cantonsdk.NewChainMetadata(
		currentOpCount,
		currentOpCount+1,
		s.chainId,
		s.proposerMcmsId,
		s.mcmsInstanceAddress,
		true,
		s.mcmsId,
	)
	s.Require().NoError(err)

	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceAddress: fmt.Sprintf("%s@%s", s.counterInstanceID, s.participant.PartyID),
		FunctionName:          "Increment",
		OperationData:         "",
		TargetCid:             s.counterCID,
		ContractIds:           []string{s.counterCID},
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	proposal := mcms.Proposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Kind:                 types.KindProposal,
			Description:          "Canton invalid signature test",
			ValidUntil:           uint32(time.Now().Add(24 * time.Hour).Unix()),
			Signatures:           []types.Signature{},
			OverridePreviousRoot: true,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				s.chainSelector: metadata,
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: s.chainSelector,
				Transaction: types.Transaction{
					To:               s.counterCID,
					Data:             []byte{},
					AdditionalFields: opAdditionalFieldsBytes,
				},
			},
		},
	}

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}
	signable, err := mcms.NewSignable(&proposal, inspectorsMap)
	s.Require().NoError(err)

	// ValidateSignatures returns (false, QuorumNotReachedError) when quorum is not met
	quorumMet, _ := signable.ValidateSignatures(ctx)
	s.Require().False(quorumMet, "quorum should not be met with zero signatures")

	// Sign with only 1 of 2 required signers (insufficient quorum)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(s.sortedSigners[0]))
	s.Require().NoError(err)

	quorumMet, _ = signable.ValidateSignatures(ctx)
	s.Require().False(quorumMet, "quorum should not be met with only 1 of 2 required signatures")

	// Attempt SetRoot with insufficient signatures -- should fail
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)
	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.LedgerServices.Command, s.submittingParty, []string{s.participant.PartyID}, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.chainSelector)
	s.Require().Error(err, "SetRoot should fail with insufficient signatures")
}
