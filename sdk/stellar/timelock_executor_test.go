package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar/mocks"

	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockExecutor_Execute(t *testing.T) {
	t.Parallel()

	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const timelockAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	invoker := mocks.NewInvoker(t)

	invoker.
		EXPECT().
		InvokeContract(
			mock.Anything,
			timelockAddr,
			"execute_batch",
			mock.Anything,
		).Return(nil, nil)

	executor := &TimelockExecutor{
		TimelockInspector: NewTimelockInspectorFromInvoker(invoker),
		invoker:           invoker,
		caller:            target,
	}

	tx, err := NewTransaction(
		target,
		"accept_ownership",
		nil,
		"Ownable",
		nil,
	)
	require.NoError(t, err)

	batch := types.BatchOperation{
		ChainSelector: stellarTestnetSelector,
		Transactions:  []types.Transaction{tx},
	}

	predecessor := common.HexToHash("0x1")
	salt := common.HexToHash("0x2")

	res, err := executor.Execute(
		context.Background(),
		batch,
		timelockAddr,
		predecessor,
		salt,
	)
	require.NoError(t, err)
	require.Equal(t, "stellar", res.ChainFamily)
}
