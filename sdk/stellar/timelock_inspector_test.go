package stellar

import (
	"context"
	"testing"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"
)

func TestTimelockInspector_ReadOperations(t *testing.T) {
	t.Parallel()

	const timelockAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const member = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	// Each role lookup first reads the member count and then reads the member.
	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"get_role_member_count",
			mock.Anything,
		).
		Return(new(scval.Uint32ToScVal(1)), nil).
		Times(4)

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"get_role_member",
			mock.Anything,
		).
		Return(new(scval.AddressToScVal(member)), nil).
		Times(4)

	boolVal := scval.BoolToScVal(true)

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"is_operation",
			mock.Anything,
		).
		Return(&boolVal, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"is_operation_pending",
			mock.Anything,
		).
		Return(&boolVal, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"is_operation_ready",
			mock.Anything,
		).
		Return(&boolVal, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"is_operation_done",
			mock.Anything,
		).
		Return(&boolVal, nil).
		Once()

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			timelockAddr,
			"get_min_delay",
			mock.Anything,
		).
		Return(new(scval.Uint64ToScVal(42)), nil).
		Once()

	inspector := NewTimelockInspectorFromInvoker(invoker)

	proposers, err := inspector.GetProposers(
		context.Background(),
		timelockAddr,
	)
	require.NoError(t, err)
	require.Equal(t, []string{member}, proposers)

	executors, err := inspector.GetExecutors(
		context.Background(),
		timelockAddr,
	)
	require.NoError(t, err)
	require.Equal(t, []string{member}, executors)

	bypassers, err := inspector.GetBypassers(
		context.Background(),
		timelockAddr,
	)
	require.NoError(t, err)
	require.Equal(t, []string{member}, bypassers)

	cancellers, err := inspector.GetCancellers(
		context.Background(),
		timelockAddr,
	)
	require.NoError(t, err)
	require.Equal(t, []string{member}, cancellers)

	var opID [32]byte

	isOp, err := inspector.IsOperation(
		context.Background(),
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isOp)

	isPending, err := inspector.IsOperationPending(
		context.Background(),
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isPending)

	isReady, err := inspector.IsOperationReady(
		context.Background(),
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isReady)

	isDone, err := inspector.IsOperationDone(
		context.Background(),
		timelockAddr,
		opID,
	)
	require.NoError(t, err)
	require.True(t, isDone)

	minDelay, err := inspector.GetMinDelay(
		context.Background(),
		timelockAddr,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(42), minDelay)

	invoker.AssertNumberOfCalls(t, "SimulateContract", 13)
}
