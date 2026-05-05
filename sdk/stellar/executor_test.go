package stellar

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/xdr"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

type recordingInvoker struct {
	lastFn string
}

func (r *recordingInvoker) InvokeContract(_ context.Context, _ string, fn string, _ []xdr.ScVal) (*xdr.ScVal, error) {
	r.lastFn = fn

	v := xdr.ScVal{}

	return &v, nil
}

func (r *recordingInvoker) SimulateContract(context.Context, string, string, []xdr.ScVal) (*xdr.ScVal, error) {
	v := xdr.ScVal{}

	return &v, nil
}

func (r *recordingInvoker) GetEvents(context.Context, string, uint32, []string) ([]protocolrpc.EventInfo, error) {
	return nil, nil
}

func TestExecutor_ExecuteOperation_routesToExecute(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.STELLAR_TESTNET.Selector)

	enc := NewEncoder(sel, 0, false)
	inv := &recordingInvoker{}
	ex := NewExecutor(enc, inv)

	ctx := context.Background()

	md := types.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
	}

	op := types.Operation{
		Transaction: types.Transaction{
			To:   "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			Data: []byte{1, 2, 3},
		},
	}

	res, err := ex.ExecuteOperation(ctx, md, 0, []common.Hash{{}}, op)
	require.NoError(t, err)
	require.Equal(t, chainsel.FamilyStellar, res.ChainFamily)
	require.Equal(t, "execute", inv.lastFn)
}

func TestExecutor_SetRoot_routesToSetRoot(t *testing.T) {
	t.Parallel()

	sel := types.ChainSelector(chainsel.STELLAR_TESTNET.Selector)

	enc := NewEncoder(sel, 1, false)
	inv := &recordingInvoker{}
	ex := NewExecutor(enc, inv)

	ctx := context.Background()

	md := types.ChainMetadata{
		StartingOpCount: 1,
		MCMAddress:      "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
	}

	sig := types.Signature{
		R: common.Hash{1},
		S: common.Hash{2},
		V: 27,
	}

	res, err := ex.SetRoot(ctx, md, []common.Hash{{}}, [32]byte{9}, 100, []types.Signature{sig})
	require.NoError(t, err)
	require.Equal(t, chainsel.FamilyStellar, res.ChainFamily)
	require.Equal(t, "set_root", inv.lastFn)
}

func TestMerkleProofFromHashes_roundTrip(t *testing.T) {
	t.Parallel()

	proof := []common.Hash{{1}, {2}}
	mp := merkleProofFromHashes(proof)
	require.Len(t, mp.Inner, 2)
	require.Equal(t, proof[0], common.Hash(mp.Inner[0]))
	require.Equal(t, proof[1], common.Hash(mp.Inner[1]))
}

func TestSignatureVecFrom_preservesComponents(t *testing.T) {
	t.Parallel()

	sigs := []types.Signature{
		{R: common.Hash{1}, S: common.Hash{2}, V: 28},
	}
	vec := signatureVecFrom(sigs)
	require.Len(t, vec.Inner, 1)
	require.Equal(t, stellarmcms.Signature{R: sigs[0].R, S: sigs[0].S, V: 28}, vec.Inner[0])
}
