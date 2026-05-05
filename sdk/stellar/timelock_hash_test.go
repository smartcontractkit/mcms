package stellar

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"

	"github.com/smartcontractkit/mcms/types"
)

func TestHashOperationBatchGolden_EmptyCalls(t *testing.T) {
	t.Parallel()
	var pred common.Hash
	var salt common.Hash
	salt[31] = 1

	got := HashOperationBatch(nil, pred, salt)
	want := common.HexToHash("0xcbfe4baa920060fc34aa65135b74b83fa81df36f6e21d90c8301c8810d2c89d9")
	require.Equal(t, want, got, "must match contracts/timelock hash_operation_batch_internal (n=0, zero pred, salt[31]=1)")
}

func TestHashOperationBatchGolden_OneCallEmptyData(t *testing.T) {
	t.Parallel()
	var to [32]byte
	for i := range to {
		to[i] = 0x11
	}
	calls := []timelockbindings.Call{{To: to, Data: nil}}
	var pred common.Hash
	var salt common.Hash
	salt[31] = 2

	got := HashOperationBatch(calls, pred, salt)
	want := common.HexToHash("0x6ead1e78e7912c0a67d23eba158933299324df465db1c2e9d5ee89aa37dea436")
	require.Equal(t, want, got)

	var concat [64]byte
	copy(concat[:32], to[:])
	copy(concat[32:], crypto.Keccak256Hash([]byte{}).Bytes())
	callH := crypto.Keccak256Hash(concat[:])
	require.Equal(t, common.HexToHash("0x0323fdea0b67062f39f74437ee69f91108a863c0a6c49271c0ff5684e4cc2c34"), callH)

	var buf []byte
	var nWord [32]byte
	binary.BigEndian.PutUint64(nWord[24:32], 1)
	buf = append(buf, nWord[:]...)
	buf = append(buf, callH[:]...)
	buf = append(buf, pred[:]...)
	buf = append(buf, salt[:]...)
	require.Equal(t, want, crypto.Keccak256Hash(buf))
}

func TestHashOperationBatchBypassZeroesPredecessor(t *testing.T) {
	t.Parallel()
	var pred common.Hash
	pred[0] = 0xab
	var salt common.Hash
	var to [32]byte
	for i := range to {
		to[i] = 1
	}
	calls := []timelockbindings.Call{
		{To: to, Data: []byte{1}},
	}

	gotSchedule := HashOperationBatch(calls, pred, salt)
	gotBypass := HashOperationBatch(calls, common.Hash{}, salt)
	require.NotEqual(t, gotSchedule, gotBypass)

	gotBypass2, err := OperationID(types.BatchOperation{
		ChainSelector: 1,
		Transactions: []types.Transaction{
			{To: strings.Repeat("01", 32), Data: []byte{1}, AdditionalFields: []byte("{}")},
		},
	}, types.TimelockActionBypass, pred, salt)
	require.NoError(t, err)
	require.Equal(t, gotBypass, gotBypass2)
}
