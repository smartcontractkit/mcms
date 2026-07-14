package mcms

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// ScheduleAndCancelTestEnv describes the chain-specific state needed to run the shared
// schedule/cancel timelock flow.
//
// Testing-only: this is intended to let tests in other packages reuse the same generic lifecycle.
type ScheduleAndCancelTestEnv struct {
	Proposal TimelockProposal
	Chains   chainwrappers.ChainAccessor
}

// ScheduleAndCancelTestHooks defines the chain-specific hooks for the shared schedule/cancel test.
//
// Testing-only: this is intended to let tests in other packages reuse the same generic lifecycle.
type ScheduleAndCancelTestHooks struct {
	Setup func(ctx context.Context, t *testing.T) (ScheduleAndCancelTestEnv, error)
	Sign  func(t *testing.T, signable *Signable)

	// PrepareConvertedProposal allows tests to tweak the converted MCMS proposal before encoders and
	// execution are built. EVM simulated backend tests use this to enable simulated-backend mode.
	PrepareConvertedProposal func(t *testing.T, proposal *Proposal)

	// DeriveCancellationMetadata allows tests to override the cancellation metadata for a chain.
	// The default runner copies the schedule metadata (only updating StartingOpCount), but some
	// chains need to change fields like AdditionalFields. For example, Aptos stores the MCMS role
	// in AdditionalFields and switches it from Proposer → Canceller for the cancel proposal.
	// If nil, the runner falls back to copying the schedule metadata as-is.
	DeriveCancellationMetadata func(t *testing.T, selector types.ChainSelector, scheduleMetadata types.ChainMetadata) (types.ChainMetadata, error)

	// WaitForTransaction is called after every SetRoot / Execute result so each chain can confirm
	// finality using its own mechanism.
	WaitForTransaction func(ctx context.Context, t *testing.T, tx types.TransactionResult)

	// AssertExtraAfterCancel is reserved for chain-specific semantic assertions that go beyond the
	// generic schedule/cancel lifecycle state checks performed by this runner.
	//
	// Examples:
	// - EVM: verify a role grant target still does not have the role after cancellation.
	// - TON: verify a grantee address is absent from the role-members list after cancellation.
	AssertExtraAfterCancel func(ctx context.Context, t *testing.T, env *ScheduleAndCancelTestEnv)
}

// RunScheduleAndCancelTest executes the shared timelock schedule/cancel lifecycle using
// chain-specific hooks for setup, signing, and transaction finality.
//
// Testing-only: this is intended to let tests in other packages reuse the same generic lifecycle.
func RunScheduleAndCancelTest(t *testing.T, hooks ScheduleAndCancelTestHooks) {
	t.Helper()

	require.NotNil(t, hooks.Setup, "ScheduleAndCancelTestHooks.Setup must not be nil")
	require.NotNil(t, hooks.Sign, "ScheduleAndCancelTestHooks.Sign must not be nil")

	ctx := t.Context()

	env, err := hooks.Setup(ctx, t)
	require.NoError(t, err)

	scheduleInspectors := runTimelockLifecycleProposal(
		ctx,
		t,
		&env.Proposal,
		env.Chains,
		hooks,
	)

	tExecutors, err := chainwrappers.BuildTimelockExecutors(env.Chains, env.Proposal.ChainMetadata, env.Proposal.Action)
	require.NoError(t, err)

	tExecutable, err := NewTimelockExecutable(ctx, &env.Proposal, tExecutors)
	require.NoError(t, err)

	for opIdx := range env.Proposal.Operations {
		assertTimelockOperationState(t, ctx, &env.Proposal, tExecutable, tExecutors, opIdx, true, true, false, false)
		assertOperationPendingState(t, tExecutable, &env.Proposal, opIdx)
		assertOperationNotReadyState(t, tExecutable, &env.Proposal, opIdx)
		assertOperationNotDoneState(t, tExecutable, &env.Proposal, opIdx)
	}

	cancelProposal, err := deriveCancellationProposal(ctx, t, &env.Proposal, scheduleInspectors, hooks)
	require.NoError(t, err)

	runTimelockLifecycleProposal(
		ctx,
		t,
		&cancelProposal,
		env.Chains,
		hooks,
	)

	for opIdx := range env.Proposal.Operations {
		assertTimelockOperationState(t, ctx, &env.Proposal, tExecutable, tExecutors, opIdx, false, false, false, false)
		assertOperationNotPendingState(t, tExecutable, &env.Proposal, opIdx)
		assertOperationNotReadyState(t, tExecutable, &env.Proposal, opIdx)
		assertOperationNotDoneState(t, tExecutable, &env.Proposal, opIdx)
	}

	if hooks.AssertExtraAfterCancel != nil {
		hooks.AssertExtraAfterCancel(ctx, t, &env)
	}
}

func runTimelockLifecycleProposal(
	ctx context.Context,
	t *testing.T,
	proposal *TimelockProposal,
	chains chainwrappers.ChainAccessor,
	hooks ScheduleAndCancelTestHooks,
) map[types.ChainSelector]sdk.Inspector {
	t.Helper()

	converters, err := chainwrappers.BuildConverters(proposal.ChainMetadata)
	require.NoError(t, err)

	mcmsProposal, _, err := proposal.Convert(ctx, converters)
	require.NoError(t, err)

	if hooks.PrepareConvertedProposal != nil {
		hooks.PrepareConvertedProposal(t, &mcmsProposal)
	}

	inspectors, err := chainwrappers.BuildInspectors(chains, mcmsProposal.ChainMetadata, proposal.Action)
	require.NoError(t, err)

	signable, err := NewSignable(&mcmsProposal, inspectors) //nolint:contextcheck //OPT-400
	require.NoError(t, err)

	require.NoError(t, signable.ValidateConfigs(ctx))

	hooks.Sign(t, signable)

	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	encoders, err := mcmsProposal.GetEncoders() //nolint:contextcheck //OPT-400
	require.NoError(t, err)

	executors, err := chainwrappers.BuildExecutors(chains, mcmsProposal.ChainMetadata, encoders, proposal.Action)
	require.NoError(t, err)

	executable, err := NewExecutable(&mcmsProposal, executors) //nolint:contextcheck //OPT-400
	require.NoError(t, err)

	tree, err := mcmsProposal.MerkleTree() //nolint:contextcheck //OPT-400
	require.NoError(t, err)

	// SetRoot is per participating chain, not per operation. A proposal may contain multiple
	// operations on the same chain, but its root only needs to be written once for that chain.
	for _, selector := range proposalChainSelectors(proposal.Operations) {
		tx, setRootErr := executable.SetRoot(ctx, selector)
		require.NoError(t, setRootErr)
		require.NotEmpty(t, tx.Hash)
		waitForTransaction(ctx, t, hooks, tx)

		root, validUntil, rootErr := inspectors[selector].GetRoot(ctx, proposal.ChainMetadata[selector].MCMAddress)
		require.NoError(t, rootErr)
		require.Equal(t, common.Hash([32]byte(tree.Root.Bytes())), root)
		require.Equal(t, proposal.ValidUntil, validUntil)
	}

	for opIdx := range proposal.Operations {
		tx, execErr := executable.Execute(ctx, opIdx)
		require.NoError(t, execErr)
		require.NotEmpty(t, tx.Hash)
		waitForTransaction(ctx, t, hooks, tx)
	}

	assertOpCounts(t, ctx, proposal, inspectors)

	return inspectors
}

func deriveCancellationProposal(
	ctx context.Context,
	t *testing.T,
	schedule *TimelockProposal,
	inspectors map[types.ChainSelector]sdk.Inspector,
	hooks ScheduleAndCancelTestHooks,
) (TimelockProposal, error) {
	t.Helper()
	cancellerMetadata := make(map[types.ChainSelector]types.ChainMetadata, len(schedule.ChainMetadata))
	for selector, metadata := range schedule.ChainMetadata {
		opCount, err := inspectors[selector].GetOpCount(ctx, metadata.MCMAddress)
		if err != nil {
			return TimelockProposal{}, err
		}

		next := metadata
		next.StartingOpCount = opCount

		// Allow chain-specific overrides (e.g., Aptos needs to swap the role in AdditionalFields
		// and use the Canceller's per-role op count instead of the Proposer's).
		if hooks.DeriveCancellationMetadata != nil {
			var err error
			next, err = hooks.DeriveCancellationMetadata(t, selector, next)
			if err != nil {
				return TimelockProposal{}, fmt.Errorf("deriving cancellation metadata for selector %d: %w", selector, err)
			}
		}

		cancellerMetadata[selector] = next
	}

	return schedule.DeriveCancellationProposal(cancellerMetadata)
}

func assertOpCounts(
	t *testing.T,
	ctx context.Context,
	proposal *TimelockProposal,
	inspectors map[types.ChainSelector]sdk.Inspector,
) {
	t.Helper()

	counts, err := proposal.OperationCounts(ctx)
	require.NoError(t, err)

	for selector, metadata := range proposal.ChainMetadata {
		opCount, getErr := inspectors[selector].GetOpCount(ctx, metadata.MCMAddress)
		require.NoError(t, getErr)
		require.Equal(t, metadata.StartingOpCount+counts[selector], opCount)
	}
}

func waitForTransaction(
	ctx context.Context,
	t *testing.T,
	hooks ScheduleAndCancelTestHooks,
	tx types.TransactionResult,
) {
	t.Helper()

	if hooks.WaitForTransaction != nil {
		hooks.WaitForTransaction(ctx, t, tx)
	}
}

func proposalChainSelectors(ops []types.BatchOperation) []types.ChainSelector {
	selectors := make([]types.ChainSelector, 0, len(ops))
	seen := make(map[types.ChainSelector]struct{}, len(ops))
	for _, op := range ops {
		if _, ok := seen[op.ChainSelector]; ok {
			continue
		}
		seen[op.ChainSelector] = struct{}{}
		selectors = append(selectors, op.ChainSelector)
	}

	return selectors
}

func assertOperationPendingState(t *testing.T, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	require.NoError(t, tExecutable.IsOperationPending(ctx, opIdx))
	selector := proposal.Operations[opIdx].ChainSelector
	require.NoError(t, tExecutable.IsChainPending(ctx, selector))
}

func assertTimelockOperationState(
	t *testing.T,
	ctx context.Context,
	proposal *TimelockProposal,
	tExecutable *TimelockExecutable,
	executors map[types.ChainSelector]sdk.TimelockExecutor,
	opIdx int,
	wantIsOperation bool,
	wantPending bool,
	wantReady bool,
	wantDone bool,
) {
	t.Helper()

	op := proposal.Operations[opIdx]
	selector := op.ChainSelector
	timelockAddress := proposal.TimelockAddresses[selector]

	opID, err := tExecutable.GetOpID(ctx, opIdx, op, selector)
	require.NoError(t, err)

	gotIsOperation, err := executors[selector].IsOperation(ctx, timelockAddress, opID)
	require.NoError(t, err)
	require.Equal(t, wantIsOperation, gotIsOperation)

	gotPending, err := executors[selector].IsOperationPending(ctx, timelockAddress, opID)
	require.NoError(t, err)
	require.Equal(t, wantPending, gotPending)

	gotReady, err := executors[selector].IsOperationReady(ctx, timelockAddress, opID)
	require.NoError(t, err)
	require.Equal(t, wantReady, gotReady)

	gotDone, err := executors[selector].IsOperationDone(ctx, timelockAddress, opID)
	require.NoError(t, err)
	require.Equal(t, wantDone, gotDone)
}

func assertOperationNotPendingState(t *testing.T, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	err := tExecutable.IsOperationPending(ctx, opIdx)
	var opErr *OperationNotPendingError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opIdx, opErr.OpIndex)
	selector := proposal.Operations[opIdx].ChainSelector
	require.ErrorAs(t, tExecutable.IsChainPending(ctx, selector), &opErr)
}

func assertOperationNotReadyState(t *testing.T, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	err := tExecutable.IsOperationReady(ctx, opIdx)
	var opErr *OperationNotReadyError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opIdx, opErr.OpIndex)
	selector := proposal.Operations[opIdx].ChainSelector
	require.ErrorAs(t, tExecutable.IsChainReady(ctx, selector), &opErr)
}

func assertOperationNotDoneState(t *testing.T, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	err := tExecutable.IsOperationDone(ctx, opIdx)
	var opErr *OperationNotDoneError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opIdx, opErr.OpIndex)
	selector := proposal.Operations[opIdx].ChainSelector
	require.ErrorAs(t, tExecutable.IsChainDone(ctx, selector), &opErr)
}
