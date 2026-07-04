//go:build e2e

package common

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type RoleConfig struct {
	Keys   []*ecdsa.PrivateKey
	Config *types.Config
}

func NewRoleConfig(t *testing.T, count int, quorum uint8) RoleConfig {
	t.Helper()

	keys := make([]*ecdsa.PrivateKey, count)
	signers := make([]common.Address, count)
	for i := range count {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		keys[i] = key
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers, func(a, b common.Address) int { return a.Cmp(b) })

	return RoleConfig{
		Keys: keys,
		Config: &types.Config{
			Quorum:  quorum,
			Signers: signers,
		},
	}
}

func (r RoleConfig) Sign(t *testing.T, signable *mcms.Signable) {
	t.Helper()

	require.LessOrEqual(t, int(r.Config.Quorum), len(r.Keys))
	for i := range int(r.Config.Quorum) {
		_, err := signable.SignAndAppend(mcms.NewPrivateKeySigner(r.Keys[i]))
		require.NoError(t, err)
	}
}

type ScheduleAndCancelTestEnv struct {
	Proposal mcms.TimelockProposal
	Chains   chainwrappers.ChainAccessor
}

type ScheduleAndCancelTestHooks struct {
	Setup func(ctx context.Context, t *testing.T) (ScheduleAndCancelTestEnv, error)
	Sign  func(t *testing.T, signable *mcms.Signable)

	PrepareConvertedProposal func(t *testing.T, proposal *mcms.Proposal)

	DeriveCancellationMetadata func(t *testing.T, selector types.ChainSelector, scheduleMetadata types.ChainMetadata) (types.ChainMetadata, error)

	WaitForTransaction func(ctx context.Context, t *testing.T, tx types.TransactionResult)

	AssertExtraAfterCancel func(ctx context.Context, t *testing.T, env *ScheduleAndCancelTestEnv)
}

func RunScheduleAndCancelTest(t *testing.T, hooks ScheduleAndCancelTestHooks) {
	t.Helper()

	require.NotNil(t, hooks.Setup, "ScheduleAndCancelTestHooks.Setup must not be nil")
	require.NotNil(t, hooks.Sign, "ScheduleAndCancelTestHooks.Sign must not be nil")

	ctx := t.Context()

	env, err := hooks.Setup(ctx, t)
	require.NoError(t, err)

	scheduleInspectors := runTimelockLifecycleProposal(ctx, t, &env.Proposal, env.Chains, hooks)

	tExecutors, err := chainwrappers.BuildTimelockExecutors(env.Chains, env.Proposal.ChainMetadata, env.Proposal.Action)
	require.NoError(t, err)

	tExecutable, err := mcms.NewTimelockExecutable(ctx, &env.Proposal, tExecutors)
	require.NoError(t, err)

	for opIdx := range env.Proposal.Operations {
		assertTimelockOperationState(t, ctx, &env.Proposal, tExecutable, tExecutors, opIdx, true, true, false, false)
		assertOperationPendingState(t, tExecutable, &env.Proposal, opIdx)
		assertOperationNotReadyState(t, tExecutable, &env.Proposal, opIdx)
		assertOperationNotDoneState(t, tExecutable, &env.Proposal, opIdx)
	}

	cancelProposal, err := deriveCancellationProposal(ctx, t, &env.Proposal, scheduleInspectors, hooks)
	require.NoError(t, err)

	runTimelockLifecycleProposal(ctx, t, &cancelProposal, env.Chains, hooks)

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
	proposal *mcms.TimelockProposal,
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

	signable, err := mcms.NewSignable(&mcmsProposal, inspectors) //nolint:contextcheck //OPT-400
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

	executable, err := mcms.NewExecutable(&mcmsProposal, executors) //nolint:contextcheck //OPT-400
	require.NoError(t, err)

	tree, err := mcmsProposal.MerkleTree() //nolint:contextcheck //OPT-400
	require.NoError(t, err)

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

	for opIdx := range mcmsProposal.Operations {
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
	schedule *mcms.TimelockProposal,
	inspectors map[types.ChainSelector]sdk.Inspector,
	hooks ScheduleAndCancelTestHooks,
) (mcms.TimelockProposal, error) {
	t.Helper()

	cancellerMetadata := make(map[types.ChainSelector]types.ChainMetadata, len(schedule.ChainMetadata))
	for selector, metadata := range schedule.ChainMetadata {
		opCount, err := inspectors[selector].GetOpCount(ctx, metadata.MCMAddress)
		if err != nil {
			return mcms.TimelockProposal{}, err
		}

		next := metadata
		next.StartingOpCount = opCount

		if hooks.DeriveCancellationMetadata != nil {
			var err error
			next, err = hooks.DeriveCancellationMetadata(t, selector, next)
			if err != nil {
				return mcms.TimelockProposal{}, fmt.Errorf("deriving cancellation metadata for selector %d: %w", selector, err)
			}
		}

		cancellerMetadata[selector] = next
	}

	return schedule.DeriveCancellationProposal(cancellerMetadata)
}

func assertOpCounts(
	t *testing.T,
	ctx context.Context,
	proposal *mcms.TimelockProposal,
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

func assertOperationPendingState(t *testing.T, tExecutable *mcms.TimelockExecutable, proposal *mcms.TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	require.NoError(t, tExecutable.IsOperationPending(ctx, opIdx))
	selector := proposal.Operations[opIdx].ChainSelector
	require.NoError(t, tExecutable.IsChainPending(ctx, selector))
}

func assertTimelockOperationState(
	t *testing.T,
	ctx context.Context,
	proposal *mcms.TimelockProposal,
	tExecutable *mcms.TimelockExecutable,
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

	if !wantIsOperation && !gotIsOperation {
		return
	}

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

func assertOperationNotPendingState(t *testing.T, tExecutable *mcms.TimelockExecutable, proposal *mcms.TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	err := tExecutable.IsOperationPending(ctx, opIdx)
	var opErr *mcms.OperationNotPendingError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opIdx, opErr.OpIndex)
	selector := proposal.Operations[opIdx].ChainSelector
	require.ErrorAs(t, tExecutable.IsChainPending(ctx, selector), &opErr)
}

func assertOperationNotReadyState(t *testing.T, tExecutable *mcms.TimelockExecutable, proposal *mcms.TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	err := tExecutable.IsOperationReady(ctx, opIdx)
	var opErr *mcms.OperationNotReadyError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opIdx, opErr.OpIndex)
	selector := proposal.Operations[opIdx].ChainSelector
	require.ErrorAs(t, tExecutable.IsChainReady(ctx, selector), &opErr)
}

func assertOperationNotDoneState(t *testing.T, tExecutable *mcms.TimelockExecutable, proposal *mcms.TimelockProposal, opIdx int) {
	t.Helper()

	ctx := t.Context()
	err := tExecutable.IsOperationDone(ctx, opIdx)
	var opErr *mcms.OperationNotDoneError
	require.ErrorAs(t, err, &opErr)
	require.Equal(t, opIdx, opErr.OpIndex)
	selector := proposal.Operations[opIdx].ChainSelector
	require.ErrorAs(t, tExecutable.IsChainDone(ctx, selector), &opErr)
}
