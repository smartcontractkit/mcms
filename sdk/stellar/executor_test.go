package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestExecutor_ExecuteOperation(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := newMockInvoker()
	inspector := NewInspectorFromInvoker(invoker)
	encoder := NewEncoder(types.ChainSelector(4894814558906953166), 1, false)

	executor, err := NewExecutor(encoder, inspector)
	require.NoError(t, err)

	op, err := NewTransaction(target, "some_func", nil, "TestContract", nil)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		MCMAddress: mcmAddr,
	}

	res, err := executor.ExecuteOperation(context.Background(), metadata, 1, []common.Hash{}, types.Operation{Transaction: op})
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	require.Len(t, invoker.calls, 1)
	require.Equal(t, "execute", invoker.calls[0].FunctionName)
	require.Equal(t, mcmAddr, invoker.calls[0].ContractID)
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := newMockInvoker()

	// Stub get_root to return a different root so SetRoot doesn't exit early
	var differentRoot [32]byte
	differentRoot[0] = 1
	rootTuple := xdr.ScVec{scval.Bytes32ToScVal(differentRoot), scval.Uint32ToScVal(0)}
	rootTuplePtr := &rootTuple
	rootValue := xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &rootTuplePtr}
	invoker.stub("get_root", &rootValue, nil)

	inspector := NewInspectorFromInvoker(invoker)
	encoder := NewEncoder(types.ChainSelector(4894814558906953166), 1, false)

	executor, err := NewExecutor(encoder, inspector)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		MCMAddress: mcmAddr,
	}

	var rootToSet [32]byte
	rootToSet[0] = 2

	res, err := executor.SetRoot(context.Background(), metadata, []common.Hash{}, rootToSet, 100, []types.Signature{})
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	// Calls should include get_root simulation and set_root execution
	require.Len(t, invoker.calls, 2)
	require.Equal(t, "get_root", invoker.calls[0].FunctionName)
	require.Equal(t, "set_root", invoker.calls[1].FunctionName)
}

func TestExecutor_SetRoot_AlreadySet(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := newMockInvoker()

	// Stub get_root to return the same root and validUntil
	var root [32]byte
	root[0] = 5
	rootTuple := xdr.ScVec{scval.Bytes32ToScVal(root), scval.Uint32ToScVal(100)}
	rootTuplePtr := &rootTuple
	rootValue := xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &rootTuplePtr}
	invoker.stub("get_root", &rootValue, nil)

	inspector := NewInspectorFromInvoker(invoker)
	encoder := NewEncoder(types.ChainSelector(4894814558906953166), 1, false)

	executor, err := NewExecutor(encoder, inspector)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		MCMAddress: mcmAddr,
	}

	_, err = executor.SetRoot(context.Background(), metadata, []common.Hash{}, root, 100, []types.Signature{})
	require.ErrorContains(t, err, "root already set")
}
