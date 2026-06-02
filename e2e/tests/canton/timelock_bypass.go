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

// TimelockBypassTestSuite tests the bypass timelock flow:
// execute immediately without scheduling, bypassing the timelock delay.
// Uses self-dispatch (UpdateMinDelay on MCMS itself) to avoid external contract ID issues.
type TimelockBypassTestSuite struct {
	mcmsExecutorSetup
}

// encodeMinDelay encodes a delay in seconds as a 16-char hex string for UpdateMinDelay operationData.
// The Canton MCMS contract decodes this via decodeInt64At and converts to RelTime via `seconds`.
func encodeMinDelay(seconds int64) string {
	return fmt.Sprintf("%016x", seconds)
}

// TestTimelockBypass executes a batch immediately via bypasser role, skipping timelock delay.
// Uses self-dispatch (UpdateMinDelay) since external contract execution requires additional SDK support.
func (s *TimelockBypassTestSuite) TestTimelockBypass() {
	ctx := s.T().Context()

	// Use bypasser role
	bypasserInspector := cantonsdk.NewInspector(s.participant.LedgerServices.State, []string{s.participant.PartyID}, cantonsdk.TimelockRoleBypasser)
	currentOpCount, err := bypasserInspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err, "get current op count")

	// Bypasser metadata
	bypasserMcmsId := fmt.Sprintf("%s@%s-bypasser", s.mcmsId, s.participant.PartyID)
	metadata, err := cantonsdk.NewChainMetadata(
		currentOpCount,
		// currentOpCount+1,
		s.chainId,
		bypasserMcmsId,
		s.mcmsInstanceAddress,
		// false,
		s.mcmsId,
	)
	s.Require().NoError(err)

	validUntil := uint32(time.Now().Add(24 * time.Hour).Unix())

	// Batch operation: UpdateMinDelay on MCMS itself (self-dispatch, no external contracts needed)
	// Use mcmsId@partyId format for self-dispatch target
	mcmsTargetInstanceAddr := fmt.Sprintf("%s@%s", s.mcmsId, s.participant.PartyID)
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceAddress: mcmsTargetInstanceAddr,
		FunctionName:          "UpdateMinDelay",
		OperationData:         encodeMinDelay(5), // Set minDelay to 5 seconds
		TargetCid:             "",                // Self-dispatch, no external target CID
		ContractIds:           []string{},        // No external contracts needed
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	bop := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions: []types.Transaction{{
			To:               mcmsTargetInstanceAddr,
			Data:             []byte{},
			AdditionalFields: opAdditionalFieldsBytes,
		}},
	}

	// Build bypass proposal - no delay needed since bypasser executes immediately
	bypassProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(validUntil).
		SetDescription("Canton timelock - bypass UpdateMinDelay").
		AddTimelockAddress(s.chainSelector, s.mcmsInstanceAddress).
		AddChainMetadata(s.chainSelector, metadata).
		SetAction(types.TimelockActionBypass).
		AddOperation(bop).
		Build()
	s.Require().NoError(err)

	// Convert to MCMS proposal (generates BypasserExecuteBatch choice)
	converter := cantonsdk.NewTimelockConverter()
	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: converter,
	}
	proposal, _, err := bypassProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	// Sign proposal
	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: bypasserInspector,
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

	// Set root and execute immediately (no timelock wait needed)
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)
	executor, err := cantonsdk.NewExecutor(encoder, bypasserInspector, s.participant.LedgerServices.Command, s.submittingParty, []string{s.participant.PartyID}, cantonsdk.TimelockRoleBypasser)
	s.Require().NoError(err)
	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txSetRoot.Hash)

	// Execute bypass operation - this executes the batch immediately
	for i := range proposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		s.Require().NoError(execErr, "execute bypass operation %d", i)
	}

	// Verify: op count increased (bypasser executes immediately, no TimelockExecutable needed)
	postOpCount, err := bypasserInspector.GetOpCount(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)
	s.Require().Equal(currentOpCount+1, postOpCount, "op count should increment after bypass execute")

	// Verify: minDelay was updated to 5 seconds (5_000_000 microseconds)
	timelockInspector := cantonsdk.NewTimelockInspector(s.participant.LedgerServices.Command, s.participant.LedgerServices.State, s.submittingParty, []string{s.participant.PartyID})
	minDelay, err := timelockInspector.GetMinDelay(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)
	s.Require().Equal(uint64(5), minDelay, "minDelay should be 5 seconds after UpdateMinDelay bypass")
}
