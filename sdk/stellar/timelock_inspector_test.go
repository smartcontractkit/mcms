package stellar

import (
	"testing"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"
)

func TestTimelockInspector_ReadOperations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	const timelockAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const member = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	invoker.EXPECT().SimulateContract(
		mock.Anything,
		timelockAddr,
		"get_role_member_count",
		mock.Anything,
	).Return(new(scval.Uint32ToScVal(1)), nil).Times(4)

	// Return a fresh address ScVal for every invocation.
	invoker.EXPECT().SimulateContract(
		mock.Anything,
		timelockAddr,
		"get_role_member",
		mock.Anything,
	).
		Return(new(scval.AddressToScVal(member)), nil).
		Times(4)

	// Return a fresh bool for each operation query.
	for _, fn := range []string{
		"is_operation",
		"is_operation_pending",
		"is_operation_ready",
		"is_operation_done",
	} {
		invoker.EXPECT().SimulateContract(
			mock.Anything,
			timelockAddr,
			fn,
			mock.Anything,
		).
			Return(new(scval.BoolToScVal(true)), nil)

		invoker.EXPECT().SimulateContract(
			mock.Anything,
			timelockAddr,
			"get_min_delay",
			mock.Anything,
		).
			Return(new(scval.Uint64ToScVal(42)), nil)
	}
	inspector := NewTimelockInspectorFromInvoker(invoker)

	proposers, err := inspector.GetProposers(ctx, timelockAddr)
	require.NoError(t, err)
	require.Equal(t, []string{member}, proposers)

	executors, err := inspector.GetExecutors(ctx, timelockAddr)
	require.NoError(t, err)
	require.Equal(t, []string{member}, executors)

	bypassers, err := inspector.GetBypassers(ctx, timelockAddr)
	require.NoError(t, err)
	require.Equal(t, []string{member}, bypassers)

	cancellers, err := inspector.GetCancellers(ctx, timelockAddr)
	require.NoError(t, err)
	require.Equal(t, []string{member}, cancellers)

	var opID [32]byte

	isOp, err := inspector.IsOperation(
		ctx,
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isOp)

	isPending, err := inspector.IsOperationPending(
		ctx,
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isPending)

	isReady, err := inspector.IsOperationReady(
		ctx,
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isReady)

	isDone, err := inspector.IsOperationDone(
		ctx,
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isDone)

	minDelay, err := inspector.GetMinDelay(
		ctx,
		timelockAddr,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(42), minDelay)
}
