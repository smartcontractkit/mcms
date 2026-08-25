package stellar

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverter_ConvertBatchToChainOperations(t *testing.T) {
	t.Parallel()

	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const timelockAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	converter := NewTimelockConverter()

	tx, err := NewTransaction(target, "accept_ownership", nil, "Ownable", nil)
	require.NoError(t, err)

	batch := types.BatchOperation{
		ChainSelector: stellarTestnetSelector,
		Transactions:  []types.Transaction{tx},
	}

	predecessor := common.HexToHash("0x1")
	salt := common.HexToHash("0x2")
	delay := types.Duration{Duration: 10 * time.Second}

	// Schedule
	ops, opID, err := converter.ConvertBatchToChainOperations(
		t.Context(),
		types.ChainMetadata{},
		batch,
		timelockAddr,
		mcmAddr,
		delay,
		types.TimelockActionSchedule,
		predecessor,
		salt,
	)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	require.NotEqual(t, common.Hash{}, opID)
	require.Equal(t, timelockAddr, ops[0].Transaction.To)

	payload, err := DecodeSorobanInvokePayload(ops[0].Transaction.Data)
	require.NoError(t, err)
	require.Equal(t, "schedule_batch", payload.Function)

	// Cancel
	ops, _, err = converter.ConvertBatchToChainOperations(
		t.Context(),
		types.ChainMetadata{},
		batch,
		timelockAddr,
		mcmAddr,
		delay,
		types.TimelockActionCancel,
		predecessor,
		salt,
	)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	payload, err = DecodeSorobanInvokePayload(ops[0].Transaction.Data)
	require.NoError(t, err)
	require.Equal(t, "cancel", payload.Function)

	// Bypass
	ops, _, err = converter.ConvertBatchToChainOperations(
		t.Context(),
		types.ChainMetadata{},
		batch,
		timelockAddr,
		mcmAddr,
		delay,
		types.TimelockActionBypass,
		predecessor,
		salt,
	)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	payload, err = DecodeSorobanInvokePayload(ops[0].Transaction.Data)
	require.NoError(t, err)
	require.Equal(t, "bypasser_execute_batch", payload.Function)
}

func TestTimelockConverter_OperationID(t *testing.T) {
	t.Parallel()

	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	tx, err := NewTransaction(target, "accept_ownership", nil, "Ownable", nil)
	require.NoError(t, err)

	batch := types.BatchOperation{
		ChainSelector: stellarTestnetSelector,
		Transactions:  []types.Transaction{tx},
	}

	predecessor := common.HexToHash("0x1")
	salt := common.HexToHash("0x2")

	id, err := OperationID(batch, types.TimelockActionSchedule, predecessor, salt)
	require.NoError(t, err)
	require.NotEqual(t, common.Hash{}, id)

	// Test empty batch error
	_, err = OperationID(types.BatchOperation{}, types.TimelockActionSchedule, predecessor, salt)
	require.Error(t, err)
}
