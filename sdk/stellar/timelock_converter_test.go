package stellar

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverter_ScheduleAndOperationID(t *testing.T) {
	t.Parallel()
	tl := strings.Repeat("c", 64)
	mcm := strings.Repeat("b", 64)
	md := types.ChainMetadata{
		MCMAddress:      mcm,
		StartingOpCount: 0,
		AdditionalFields: mustJSON(t, map[string]string{
			"timelockProposer": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		}),
	}
	bop := types.BatchOperation{
		ChainSelector: 1,
		Transactions: []types.Transaction{
			{
				OperationMetadata: types.OperationMetadata{Tags: []string{"t1"}},
				To:                strings.Repeat("a", 64),
				Data:              []byte{0xde, 0xad},
				AdditionalFields:  []byte("{}"),
			},
		},
	}

	conv := NewTimelockConverter()
	opIDWant, err := OperationID(bop, types.TimelockActionSchedule, common.Hash{}, common.Hash{31: 3})
	require.NoError(t, err)

	ops, opIDGot, err := conv.ConvertBatchToChainOperations(
		t.Context(),
		md,
		bop,
		tl,
		mcm,
		types.NewDuration(100*time.Second),
		types.TimelockActionSchedule,
		common.Hash{},
		common.Hash{31: 3},
	)
	require.NoError(t, err)
	require.Equal(t, opIDWant, opIDGot)
	require.Len(t, ops, 1)
	require.Equal(t, tl, ops[0].Transaction.To)
	require.NotEmpty(t, ops[0].Transaction.Data)
	require.Equal(t, "RBACTimelock", ops[0].Transaction.ContractType)
	require.Equal(t, []string{"t1"}, ops[0].Transaction.Tags)
}

func TestTimelockConverter_MissingProposer(t *testing.T) {
	t.Parallel()
	conv := NewTimelockConverter()
	md := types.ChainMetadata{
		MCMAddress:       strings.Repeat("b", 64),
		AdditionalFields: mustJSON(t, map[string]string{}),
	}
	bop := types.BatchOperation{
		ChainSelector: 1,
		Transactions: []types.Transaction{
			{To: strings.Repeat("a", 64), Data: []byte{1}, AdditionalFields: []byte("{}")},
		},
	}
	_, _, err := conv.ConvertBatchToChainOperations(
		t.Context(), md, bop,
		strings.Repeat("c", 64),
		strings.Repeat("b", 64),
		types.NewDuration(time.Second),
		types.TimelockActionSchedule,
		common.Hash{}, common.Hash{},
	)
	require.ErrorContains(t, err, "timelockProposer")
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}
