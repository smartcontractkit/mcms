package stellar

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestConfigToSetConfigInputs_FlattensNestedConfig(t *testing.T) {
	t.Parallel()
	cfg := &types.Config{
		Quorum: 2,
		Signers: []common.Address{
			common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"),
		},
		GroupSigners: []types.Config{{
			Quorum: 1,
			Signers: []common.Address{
				common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
			},
		}},
	}
	addrs, groups, gq, gp, err := ConfigToSetConfigInputs(cfg)
	require.NoError(t, err)

	// Two signers total, sorted by address (0xAA.. < 0xBB..).
	require.Len(t, addrs.Inner, 2)
	require.Equal(t, common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA").Bytes(), addrs.Inner[0][12:])
	require.Equal(t, common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB").Bytes(), addrs.Inner[1][12:])

	// Signer 0xAAAA is in the child group (index 1); 0xBBBB is in root (index 0).
	// After sorting by address, groups come out [1, 0].
	require.Equal(t, []uint32{1, 0}, groups.Inner[:])
	require.Equal(t, uint8(2), gq[0])
	require.Equal(t, uint8(1), gq[1])
	require.Equal(t, uint8(0), gp[1]) // child group parent is root
}

func TestNewBatchOperationUsesCanonicalSorobanPayload(t *testing.T) {
	t.Parallel()
	const target = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"

	batch, err := NewBatchOperation(
		types.ChainSelector(10001),
		target,
		"accept_ownership",
		nil,
		"StellarForwarder",
		[]string{"ownership"},
	)
	require.NoError(t, err)
	require.Len(t, batch.Transactions, 1)
	require.Equal(t, target, batch.Transactions[0].To)
	require.JSONEq(t, `{"family":"stellar","encodingVersion":1}`, string(batch.Transactions[0].AdditionalFields))

	payload, err := DecodeSorobanInvokePayload(batch.Transactions[0].Data)
	require.NoError(t, err)
	require.Equal(t, "accept_ownership", payload.Function)
	require.Empty(t, payload.Args)
	require.NotEmpty(t, payload.ArgsXDR, "empty args are represented by an XDR ScVec")
}

func TestNewTransactionRejectsInvalidTarget(t *testing.T) {
	t.Parallel()
	_, err := NewTransaction("not-a-contract", "accept_ownership", nil, "", nil)
	require.Error(t, err)
}

func TestNewTransactionAcceptsRawAndStrKeyContractIDs(t *testing.T) {
	t.Parallel()
	const strKey = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	const raw = "3f0c34bf93ad0d9971d04ccc90f705511c838aad9734a4a2fb0d7a03fc7fe89a"
	for _, target := range []string{strKey, raw, "0x" + raw} {
		tx, err := NewTransaction(target, "accept_ownership", nil, "Ownable", nil)
		require.NoError(t, err)
		require.Equal(t, target, tx.To)
	}
}
