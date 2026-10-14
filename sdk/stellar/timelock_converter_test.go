package stellar

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stellar/go-stellar-sdk/xdr"
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

// TestTimelockConverter_MultiTransactionBatch covers batches with more than
// one transaction — the shape produced by governed forwarder allow-list
// changes — which single-transaction tests leave unexercised.
func TestTimelockConverter_MultiTransactionBatch(t *testing.T) {
	t.Parallel()

	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const timelockAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const mcmAddr = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	converter := NewTimelockConverter()

	functions := []string{"add_forwarder", "accept_ownership", "set_config"}
	transactions := make([]types.Transaction, 0, len(functions))
	for _, fn := range functions {
		tx, err := NewTransaction(target, fn, nil, "StellarForwarder", nil)
		require.NoError(t, err)
		transactions = append(transactions, tx)
	}

	batch := types.BatchOperation{
		ChainSelector: stellarTestnetSelector,
		Transactions:  transactions,
	}

	predecessor := common.HexToHash("0x1")
	salt := common.HexToHash("0x2")
	delay := types.Duration{Duration: 10 * time.Second}

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

	payload, err := DecodeSorobanInvokePayload(ops[0].Transaction.Data)
	require.NoError(t, err)
	require.Equal(t, "schedule_batch", payload.Function)
	require.Len(t, payload.Args, 5)

	// args[1] is the calls struct {inner: Vec<call>}; every transaction of the
	// batch must appear, in order.
	callsStruct := payload.Args[1]
	require.NotNil(t, callsStruct.Map)
	entries := **callsStruct.Map
	require.Len(t, entries, 1)
	require.NotNil(t, entries[0].Val.Vec)
	calls := **entries[0].Val.Vec
	require.Len(t, calls, len(functions))

	for i, call := range calls {
		require.NotNil(t, call.Map)
		fields := map[string]xdr.ScVal{}
		for _, entry := range **call.Map {
			sym, symErr := entry.Key.GetSym()
			require.True(t, symErr)
			fields[string(sym)] = entry.Val
		}
		sym, ok := fields["function"].GetSym()
		require.True(t, ok)
		require.Equal(t, functions[i], string(sym))
	}

	// The operation ID must match the standalone hash and be order-sensitive.
	standaloneID, err := OperationID(batch, types.TimelockActionSchedule, predecessor, salt)
	require.NoError(t, err)
	require.Equal(t, standaloneID, opID)

	reversed := types.BatchOperation{
		ChainSelector: stellarTestnetSelector,
		Transactions:  []types.Transaction{transactions[2], transactions[1], transactions[0]},
	}
	reversedID, err := OperationID(reversed, types.TimelockActionSchedule, predecessor, salt)
	require.NoError(t, err)
	require.NotEqual(t, opID, reversedID)

	// Bypass must carry the same multi-call vector.
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
	require.NotNil(t, payload.Args[1].Map)
	require.NotNil(t, (**payload.Args[1].Map)[0].Val.Vec)
	require.Len(t, **(**payload.Args[1].Map)[0].Val.Vec, len(functions))
}

// TestTimelockConverter_GoldenVector mirrors the timelock section of the
// normative cross-language fixture
// chainlink-stellar/contracts/mcms/testdata/stellar_golden_vectors.json,
// which the Rust contract asserts in contracts/timelock/src/encoding.rs
// (golden_vector_matches_normative_fixture). The MCMS-leaf half of the same
// fixture is mirrored in current_encoding_test.go; this pins the timelock
// half so Go/Rust hash drift fails a unit test on either side.
func TestTimelockConverter_GoldenVector(t *testing.T) {
	t.Parallel()

	// target = sequential bytes 0x40..0x5f, as in the fixture.
	const target = "0x404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f"

	tx, err := NewTransaction(target, "schedule_batch", nil, "RBACTimelock", nil)
	require.NoError(t, err)

	payload, err := DecodeSorobanInvokePayload(tx.Data)
	require.NoError(t, err)
	// args_xdr of an empty Vec<Val>, per the fixture's call_preimage_hex.
	require.Equal(t, "000000100000000100000000", common.Bytes2Hex(payload.ArgsXDR))

	targetHash, err := AddressHash(target)
	require.NoError(t, err)

	callHash := hashTimelockCall([32]byte(targetHash), payload.Function, payload.ArgsXDR)
	require.Equal(
		t,
		common.HexToHash("0x18d3e91726884ecc896974458f55e165731883b27c43c3cc7bb0f89f0447dd66"),
		callHash,
	)

	batch := types.BatchOperation{
		ChainSelector: stellarTestnetSelector,
		Transactions:  []types.Transaction{tx},
	}
	predecessor := common.Hash{}
	salt := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	opID, err := OperationID(batch, types.TimelockActionSchedule, predecessor, salt)
	require.NoError(t, err)
	require.Equal(
		t,
		common.HexToHash("0x1ce67c588ad5f00c9add1f334b272d23f40a8f9155ab6f9cff2d8c35ee16e79e"),
		opID,
	)
}
