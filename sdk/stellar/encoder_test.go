package stellar

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

// stellar-testnet selector from chain-selectors selectors_stellar.yml
const stellarTestnetSelector types.ChainSelector = 4894814558906953166

func TestEncoder_HashMetadataAndOperation(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(stellarTestnetSelector, 1, false)

	chainNet, err := ChainNetworkID(stellarTestnetSelector)
	require.NoError(t, err)

	mcm := "cee0302d59844d32bdca915c8203dd44b33fbb7edc19051ea37abedf28ecd472"
	require.Equal(t, chainNet.Hex()[2:], mcm, "sanity: selector maps to expected network id hex")

	metaAddr := "00000000000000000000000000000000000000000000000000000000000000aa"
	toAddr := "00000000000000000000000000000000000000000000000000000000000000bb"

	metaHashManual, err := HashRootMetadata(
		domainMetaStellar,
		chainNet,
		hashBytes(t, metaAddr),
		0,
		1,
		false,
	)
	require.NoError(t, err)

	md := types.ChainMetadata{
		StartingOpCount:  0,
		MCMAddress:       "0x" + metaAddr,
		AdditionalFields: nil,
	}
	metaHashEnc, err := enc.HashMetadata(md)
	require.NoError(t, err)
	require.Equal(t, metaHashManual, metaHashEnc)

	op := types.Operation{
		ChainSelector: stellarTestnetSelector,
		Transaction: types.Transaction{
			To:               "0x" + toAddr,
			Data:             []byte{0xde, 0xad},
			AdditionalFields: nil,
		},
	}
	opHashManual, err := HashStellarOp(
		domainOpStellar,
		chainNet,
		hashBytes(t, metaAddr),
		0,
		hashBytes(t, toAddr),
		[32]byte{},
		op.Transaction.Data,
	)
	require.NoError(t, err)

	opHashEnc, err := enc.HashOperation(0, md, op)
	require.NoError(t, err)
	require.Equal(t, opHashManual, opHashEnc)
}

func TestParseContractID_Strkey(t *testing.T) {
	t.Parallel()
	// Vector from github.com/stellar/go/strkey decode_test ("Contract" case).
	const sample = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	want := common.HexToHash("0x3f0c34bf93ad0d9971d04ccc90f705511c838aad9734a4a2fb0d7a03fc7fe89a")
	got, err := ParseContractID(sample)
	require.NoError(t, err)
	require.Equal(t, want, common.Hash(got))
	round, err := ParseContractID("0x" + common.Bytes2Hex(got[:]))
	require.NoError(t, err)
	require.Equal(t, got, round)
}

func TestEncoder_PostOpCountOverflow(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(stellarTestnetSelector, 1<<40, false)
	_, err := enc.HashMetadata(types.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x" + strings.Repeat("00", stellarContractIDBytes),
	})
	require.ErrorIs(t, err, ErrUint40Overflow)
}
