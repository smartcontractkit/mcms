package stellar_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms/sdk/stellar"
	stellarmocks "github.com/smartcontractkit/mcms/sdk/stellar/mocks"
)

func TestTimelockInspector_IsInitialized_RoleMembership(t *testing.T) {
	t.Parallel()

	roles := []string{"ADMIN", "PROPOSER", "CANCELLER", "BYPASSER"}

	for activeIndex, activeRole := range roles {
		t.Run(activeRole, func(t *testing.T) {
			t.Parallel()

			//nolint:gosec // G115 conversion safe
			address := testContractID(t, byte(100+activeIndex))
			invoker := stellarmocks.NewInvoker(t)

			for index := 0; index <= activeIndex; index++ {
				count := uint32(0)
				if index == activeIndex {
					count = 1
				}

				invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count", []xdr.ScVal{scval.SymbolToScVal(roles[index])}).Return(new(scval.Uint32ToScVal(count)), nil).Once()
			}

			inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

			initialized, err := inspector.IsInitialized(t.Context(), address)
			require.NoError(t, err)
			require.True(t, initialized)

			invoker.AssertNotCalled(t, "SimulateContract", mock.Anything, address, "get_min_delay", mock.Anything)
		})
	}
}

func TestTimelockInspector_IsInitialized_MinDelay(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 110)
	invoker := stellarmocks.NewInvoker(t)

	for _, role := range []string{"ADMIN", "PROPOSER", "CANCELLER", "BYPASSER"} {
		invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count", []xdr.ScVal{scval.SymbolToScVal(role)}).Return(new(scval.Uint32ToScVal(0)), nil).Once()
	}

	invoker.On("SimulateContract", mock.Anything, address, "get_min_delay", []xdr.ScVal{}).Return(new(scval.Uint64ToScVal(1)), nil).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	initialized, err := inspector.IsInitialized(t.Context(), address)
	require.NoError(t, err)
	require.True(t, initialized)
}

func TestTimelockInspector_IsInitialized_EmptyState(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 111)
	invoker := stellarmocks.NewInvoker(t)

	for _, role := range []string{"ADMIN", "PROPOSER", "CANCELLER", "BYPASSER"} {
		invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count", []xdr.ScVal{scval.SymbolToScVal(role)}).Return(new(scval.Uint32ToScVal(0)), nil).Once()
	}

	invoker.On("SimulateContract", mock.Anything, address, "get_min_delay", []xdr.ScVal{}).Return(new(scval.Uint64ToScVal(0)), nil).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	initialized, err := inspector.IsInitialized(t.Context(), address)
	require.NoError(t, err)
	require.False(t, initialized)
}

func TestTimelockInspector_IsInitialized_RoleReadError(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 112)
	expectedErr := errors.New("role read failed")
	invoker := stellarmocks.NewInvoker(t)

	invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count", []xdr.ScVal{scval.SymbolToScVal("ADMIN")}).Return((*xdr.ScVal)(nil), expectedErr).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	initialized, err := inspector.IsInitialized(t.Context(), address)
	require.False(t, initialized)
	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(t, err, "ADMIN role member count")
}

func TestTimelockInspector_IsInitialized_MinDelayReadError(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 113)
	expectedErr := errors.New("minimum delay read failed")
	invoker := stellarmocks.NewInvoker(t)

	for _, role := range []string{"ADMIN", "PROPOSER", "CANCELLER", "BYPASSER"} {
		invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count", []xdr.ScVal{scval.SymbolToScVal(role)}).Return(new(scval.Uint32ToScVal(0)), nil).Once()
	}

	invoker.On("SimulateContract", mock.Anything, address, "get_min_delay", []xdr.ScVal{}).Return((*xdr.ScVal)(nil), expectedErr).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	initialized, err := inspector.IsInitialized(t.Context(), address)
	require.False(t, initialized)
	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(t, err, "minimum delay")
}

func TestTimelockInspector_IsInitialized_NilInvoker(t *testing.T) {
	t.Parallel()

	inspector := stellar.NewTimelockInspectorFromInvoker(nil)

	initialized, err := inspector.IsInitialized(t.Context(), testContractID(t, 114))
	require.False(t, initialized)
	require.ErrorContains(t, err, "invoker is nil")
}

func TestTimelockInspector_IsInitialized_EmptyContractID(t *testing.T) {
	t.Parallel()

	invoker := stellarmocks.NewInvoker(t)
	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	initialized, err := inspector.IsInitialized(t.Context(), "")
	require.False(t, initialized)
	require.ErrorContains(t, err, "contract ID is empty")

	invoker.AssertNotCalled(t, "SimulateContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTimelockInspector_GetExecutors_Unsupported(t *testing.T) {
	t.Parallel()

	// No invoker expectations: GetExecutors must fail without issuing any RPC,
	// because the Stellar timelock defines no executor role.
	inspector := stellar.NewTimelockInspectorFromInvoker(stellarmocks.NewInvoker(t))

	executors, err := inspector.GetExecutors(t.Context(), testContractID(t, 120))
	require.Error(t, err)
	require.Nil(t, executors)
	require.Contains(t, err.Error(), "unsupported on Stellar")
}

func TestTimelockInspector_RoleGetters(t *testing.T) {
	t.Parallel()

	member1 := testContractID(t, 121)
	member2 := testContractID(t, 122)

	tests := []struct {
		name string
		role string
		call func(i *stellar.TimelockInspector, ctx context.Context, address string) ([]string, error)
	}{
		{
			name: "proposers",
			role: "PROPOSER",
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) ([]string, error) {
				return i.GetProposers(ctx, address)
			},
		},
		{
			name: "bypassers",
			role: "BYPASSER",
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) ([]string, error) {
				return i.GetBypassers(ctx, address)
			},
		},
		{
			name: "cancellers",
			role: "CANCELLER",
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) ([]string, error) {
				return i.GetCancellers(ctx, address)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			address := testContractID(t, 125)
			invoker := stellarmocks.NewInvoker(t)

			invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count",
				[]xdr.ScVal{scval.SymbolToScVal(tc.role)}).Return(new(scval.Uint32ToScVal(2)), nil).Once()
			invoker.On("SimulateContract", mock.Anything, address, "get_role_member",
				[]xdr.ScVal{scval.SymbolToScVal(tc.role), scval.Uint32ToScVal(0)}).Return(new(scval.AddressToScVal(member1)), nil).Once()
			invoker.On("SimulateContract", mock.Anything, address, "get_role_member",
				[]xdr.ScVal{scval.SymbolToScVal(tc.role), scval.Uint32ToScVal(1)}).Return(new(scval.AddressToScVal(member2)), nil).Once()

			inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

			members, err := tc.call(inspector, t.Context(), address)
			require.NoError(t, err)
			require.Equal(t, []string{member1, member2}, members)
		})
	}
}

func TestTimelockInspector_RoleGetter_EmptyRole(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 126)
	invoker := stellarmocks.NewInvoker(t)

	invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count",
		[]xdr.ScVal{scval.SymbolToScVal("PROPOSER")}).Return(new(scval.Uint32ToScVal(0)), nil).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	proposers, err := inspector.GetProposers(t.Context(), address)
	require.NoError(t, err)
	require.NotNil(t, proposers)
	require.Empty(t, proposers)
}

func TestTimelockInspector_RoleGetter_MemberReadError(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 127)
	invoker := stellarmocks.NewInvoker(t)

	readErr := errors.New("get_role_member failed")
	invoker.On("SimulateContract", mock.Anything, address, "get_role_member_count",
		[]xdr.ScVal{scval.SymbolToScVal("CANCELLER")}).Return(new(scval.Uint32ToScVal(1)), nil).Once()
	invoker.On("SimulateContract", mock.Anything, address, "get_role_member",
		[]xdr.ScVal{scval.SymbolToScVal("CANCELLER"), scval.Uint32ToScVal(0)}).Return(nil, readErr).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	cancellers, err := inspector.GetCancellers(t.Context(), address)
	require.ErrorIs(t, err, readErr)
	require.Nil(t, cancellers)
}

func TestTimelockInspector_IsOperationProbes(t *testing.T) {
	t.Parallel()

	var opID [32]byte
	copy(opID[:], "stellar-unit-timelock-op-id")

	tests := []struct {
		name   string
		fnName string
		result bool
		call   func(i *stellar.TimelockInspector, ctx context.Context, address string) (bool, error)
	}{
		{
			name:   "is_operation true",
			fnName: "is_operation",
			result: true,
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) (bool, error) {
				return i.IsOperation(ctx, address, opID)
			},
		},
		{
			name:   "is_operation false",
			fnName: "is_operation",
			result: false,
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) (bool, error) {
				return i.IsOperation(ctx, address, opID)
			},
		},
		{
			name:   "is_operation_pending",
			fnName: "is_operation_pending",
			result: true,
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) (bool, error) {
				return i.IsOperationPending(ctx, address, opID)
			},
		},
		{
			name:   "is_operation_ready",
			fnName: "is_operation_ready",
			result: true,
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) (bool, error) {
				return i.IsOperationReady(ctx, address, opID)
			},
		},
		{
			name:   "is_operation_done",
			fnName: "is_operation_done",
			result: false,
			call: func(i *stellar.TimelockInspector, ctx context.Context, address string) (bool, error) {
				return i.IsOperationDone(ctx, address, opID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			address := testContractID(t, 128)
			invoker := stellarmocks.NewInvoker(t)

			invoker.On("SimulateContract", mock.Anything, address, tc.fnName,
				[]xdr.ScVal{scval.Bytes32ToScVal(opID)}).Return(new(scval.BoolToScVal(tc.result)), nil).Once()

			inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

			got, err := tc.call(inspector, t.Context(), address)
			require.NoError(t, err)
			require.Equal(t, tc.result, got)
		})
	}
}

func TestTimelockInspector_IsOperationProbe_ReadError(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 129)
	invoker := stellarmocks.NewInvoker(t)

	readErr := errors.New("is_operation_pending failed")
	var opID [32]byte
	copy(opID[:], "stellar-unit-timelock-op-id")

	invoker.On("SimulateContract", mock.Anything, address, "is_operation_pending",
		[]xdr.ScVal{scval.Bytes32ToScVal(opID)}).Return(nil, readErr).Once()

	inspector := stellar.NewTimelockInspectorFromInvoker(invoker)

	pending, err := inspector.IsOperationPending(t.Context(), address, opID)
	require.ErrorIs(t, err, readErr)
	require.False(t, pending)
}

func TestTimelockInspector_IsOperationProbe_NilInvoker(t *testing.T) {
	t.Parallel()

	inspector := stellar.NewTimelockInspectorFromInvoker(nil)

	var opID [32]byte
	copy(opID[:], "stellar-unit-timelock-op-id")

	isOp, err := inspector.IsOperation(t.Context(), testContractID(t, 130), opID)
	require.False(t, isOp)
	require.ErrorContains(t, err, "invoker is nil")
}
