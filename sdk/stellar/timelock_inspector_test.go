package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/types"
)

// timelockSimInvoker stubs Soroban simulation for timelock read methods used by TimelockInspector.
type timelockSimInvoker struct {
	minDelay   uint64
	roleCounts map[string]uint32
	roleMember map[string]map[uint32]string
	opExists   map[[32]byte]bool
	opPending  map[[32]byte]bool
	opReady    map[[32]byte]bool
	opDone     map[[32]byte]bool
}

func (m *timelockSimInvoker) InvokeContract(context.Context, string, string, []xdr.ScVal) (*xdr.ScVal, error) {
	return nil, invokerNotImplementedError{}
}

func (m *timelockSimInvoker) GetEvents(context.Context, string, uint32, []string) ([]protocolrpc.EventInfo, error) {
	return nil, invokerNotImplementedError{}
}

func (m *timelockSimInvoker) SimulateContract(_ context.Context, _ string, fn string, args []xdr.ScVal) (*xdr.ScVal, error) {
	switch fn {
	case "get_min_delay":
		v := scval.Uint64ToScVal(m.minDelay)

		return &v, nil

	case "get_role_member_count":
		role, err := scval.SymbolFromScVal(args[0])
		if err != nil {
			return nil, err
		}

		n := m.roleCounts[role]
		v := scval.Uint32ToScVal(n)

		return &v, nil

	case "get_role_member":
		role, err := scval.SymbolFromScVal(args[0])
		if err != nil {
			return nil, err
		}

		idx, err := scval.Uint32FromScVal(args[1])
		if err != nil {
			return nil, err
		}

		addr := m.roleMember[role][idx]
		v := scval.AddressToScVal(addr)

		return &v, nil

	case "is_operation":
		id, err := scval.Bytes32FromScVal(args[0])
		if err != nil {
			return nil, err
		}

		b := m.opExists[id]
		v := scval.BoolToScVal(b)

		return &v, nil

	case "is_operation_pending":
		id, err := scval.Bytes32FromScVal(args[0])
		if err != nil {
			return nil, err
		}

		b := m.opPending[id]
		v := scval.BoolToScVal(b)

		return &v, nil

	case "is_operation_ready":
		id, err := scval.Bytes32FromScVal(args[0])
		if err != nil {
			return nil, err
		}

		b := m.opReady[id]
		v := scval.BoolToScVal(b)

		return &v, nil

	case "is_operation_done":
		id, err := scval.Bytes32FromScVal(args[0])
		if err != nil {
			return nil, err
		}

		b := m.opDone[id]
		v := scval.BoolToScVal(b)

		return &v, nil

	default:
		return nil, invokerNotImplementedError{}
	}
}

func TestTimelockInspector_rolesAndOps(t *testing.T) {
	t.Parallel()

	execAddr := "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	inv := &timelockSimInvoker{
		minDelay: 42,
		roleCounts: map[string]uint32{
			timelockRoleProposer: 1,
		},
		roleMember: map[string]map[uint32]string{
			timelockRoleProposer: {0: execAddr},
		},
	}
	var opKey [32]byte
	opKey[0] = 1
	inv.opExists = map[[32]byte]bool{opKey: true}
	inv.opPending = map[[32]byte]bool{opKey: true}
	inv.opReady = map[[32]byte]bool{}
	inv.opDone = map[[32]byte]bool{}

	tl := stringsRepeatHexAddr('c')
	ins := NewTimelockInspector(inv)

	ctx := t.Context()
	delay, err := ins.GetMinDelay(ctx, tl)
	require.NoError(t, err)
	require.Equal(t, uint64(42), delay)

	proposers, err := ins.GetProposers(ctx, tl)
	require.NoError(t, err)
	require.Equal(t, []string{execAddr}, proposers)

	_, err = ins.GetExecutors(ctx, tl)
	require.NoError(t, err)

	opID := [32]byte{1}
	ok, err := ins.IsOperation(ctx, tl, opID)
	require.NoError(t, err)
	require.True(t, ok)

	pending, err := ins.IsOperationPending(ctx, tl, opID)
	require.NoError(t, err)
	require.True(t, pending)
}

func TestTimelockExecutor_ExecuteRequiresCaller(t *testing.T) {
	t.Parallel()
	e := NewTimelockExecutor(&timelockSimInvoker{}, "")
	_, err := e.Execute(t.Context(), types.BatchOperation{
		ChainSelector: 1,
		Transactions: []types.Transaction{
			{To: stringsRepeatHexAddr('a'), Data: []byte{1}, AdditionalFields: []byte("{}")},
		},
	}, stringsRepeatHexAddr('b'), common.Hash{}, common.Hash{})
	require.ErrorContains(t, err, "executor caller")
}

func stringsRepeatHexAddr(c byte) string {
	const n = 64
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}

	return string(b)
}
