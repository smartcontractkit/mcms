package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"

	"github.com/smartcontractkit/mcms/types"
)

func TestExecutor_ExecuteOperation(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	invoker.
		On(
			"InvokeContract",
			mock.Anything,
			mcmAddr,
			"execute",
			mock.Anything,
		).
		Return(nil, nil).
		Once()

	inspector := NewInspectorFromInvoker(invoker)
	encoder := NewEncoder(types.ChainSelector(4894814558906953166), 1, false)

	executor, err := NewExecutor(encoder, inspector)
	require.NoError(t, err)

	op, err := NewTransaction(
		target,
		"some_func",
		nil,
		"TestContract",
		nil,
	)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		MCMAddress: mcmAddr,
	}

	res, err := executor.ExecuteOperation(
		context.Background(),
		metadata,
		1,
		[]common.Hash{},
		types.Operation{Transaction: op},
	)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	invoker.AssertNumberOfCalls(t, "InvokeContract", 1)
	invoker.AssertCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mcmAddr,
		"execute",
		mock.Anything,
	)
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	// Return a different root so SetRoot doesn't exit early.
	var differentRoot [32]byte
	differentRoot[0] = 1

	rootTuple := xdr.ScVec{
		scval.Bytes32ToScVal(differentRoot),
		scval.Uint32ToScVal(0),
	}
	rootTuplePtr := &rootTuple
	rootValue := xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &rootTuplePtr,
	}

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			mcmAddr,
			"get_root",
			mock.Anything,
		).
		Return(&rootValue, nil).
		Once()

	invoker.
		On(
			"InvokeContract",
			mock.Anything,
			mcmAddr,
			"set_root",
			mock.Anything,
		).
		Return(nil, nil).
		Once()

	inspector := NewInspectorFromInvoker(invoker)
	encoder := NewEncoder(types.ChainSelector(4894814558906953166), 1, false)

	executor, err := NewExecutor(encoder, inspector)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		MCMAddress: mcmAddr,
	}

	var rootToSet [32]byte
	rootToSet[0] = 2

	res, err := executor.SetRoot(
		context.Background(),
		metadata,
		[]common.Hash{},
		rootToSet,
		100,
		[]types.Signature{},
	)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)

	invoker.AssertNumberOfCalls(t, "SimulateContract", 1)
	invoker.AssertNumberOfCalls(t, "InvokeContract", 1)

	invoker.AssertCalled(
		t,
		"SimulateContract",
		mock.Anything,
		mcmAddr,
		"get_root",
		mock.Anything,
	)

	invoker.AssertCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mcmAddr,
		"set_root",
		mock.Anything,
	)
}

func TestExecutor_SetRoot_AlreadySet(t *testing.T) {
	t.Parallel()

	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	// Return the same root and validUntil so SetRoot exits early.
	var root [32]byte
	root[0] = 5

	rootTuple := xdr.ScVec{
		scval.Bytes32ToScVal(root),
		scval.Uint32ToScVal(100),
	}
	rootTuplePtr := &rootTuple
	rootValue := xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &rootTuplePtr,
	}

	invoker.
		On(
			"SimulateContract",
			mock.Anything,
			mcmAddr,
			"get_root",
			mock.Anything,
		).
		Return(&rootValue, nil).
		Once()

	inspector := NewInspectorFromInvoker(invoker)
	encoder := NewEncoder(types.ChainSelector(4894814558906953166), 1, false)

	executor, err := NewExecutor(encoder, inspector)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		MCMAddress: mcmAddr,
	}

	_, err = executor.SetRoot(
		context.Background(),
		metadata,
		[]common.Hash{},
		root,
		100,
		[]types.Signature{},
	)
	require.ErrorContains(t, err, "root already set")

	// We should only read the current root. No transaction should be submitted.
	invoker.AssertNumberOfCalls(t, "SimulateContract", 1)
	invoker.AssertNotCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}
