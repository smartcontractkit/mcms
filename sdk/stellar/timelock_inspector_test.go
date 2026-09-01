package stellar_test

import (
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
