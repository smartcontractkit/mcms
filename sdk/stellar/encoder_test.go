package stellar

import (
	"encoding/json"
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

	chainNet, err := chainNetworkID(stellarTestnetSelector)
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

func Test_parseContractID_Strkey(t *testing.T) {
	t.Parallel()
	// Vector from github.com/stellar/go-stellar-sdk/strkey decode_test ("Contract" case).
	const sample = "CA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA"
	want := common.HexToHash("0x3f0c34bf93ad0d9971d04ccc90f705511c838aad9734a4a2fb0d7a03fc7fe89a")
	got, err := parseContractID(sample)
	require.NoError(t, err)
	require.Equal(t, want, common.Hash(got))
	round, err := parseContractID("0x" + common.Bytes2Hex(got[:]))
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

func TestEncoder_HashMetadata_StartingOpCountOverflow(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(stellarTestnetSelector, 0, false)
	_, err := enc.HashMetadata(types.ChainMetadata{
		StartingOpCount: uint40MaxExclusive,
		MCMAddress:      "0x" + strings.Repeat("00", stellarContractIDBytes),
	})
	require.ErrorIs(t, err, ErrUint40Overflow)
}

func TestEncoder_HashMetadata_InvalidChainSelector(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(types.ChainSelector(0), 1, false)
	_, err := enc.HashMetadata(types.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x" + strings.Repeat("00", stellarContractIDBytes),
	})
	require.ErrorContains(t, err, "HashMetadata: chain id:")
	require.ErrorContains(t, err, "selector 0")
}

func TestEncoder_HashOperation_InvalidChainSelector(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(types.ChainSelector(0), 1, false)
	md := types.ChainMetadata{
		MCMAddress: "0x" + strings.Repeat("11", stellarContractIDBytes),
	}
	op := types.Operation{
		Transaction: types.Transaction{
			To: "0x" + strings.Repeat("22", stellarContractIDBytes),
		},
	}
	_, err := enc.HashOperation(0, md, op)
	require.ErrorContains(t, err, "HashOperation: chain id:")
	require.ErrorContains(t, err, "selector 0")
}

func Test_parseValueWord(t *testing.T) {
	t.Parallel()

	t.Run("empty raw", func(t *testing.T) {
		t.Parallel()
		got, err := parseValueWord(nil)
		require.NoError(t, err)
		require.Equal(t, [32]byte{}, got)
	})

	t.Run("empty json object", func(t *testing.T) {
		t.Parallel()
		got, err := parseValueWord(json.RawMessage(`{}`))
		require.NoError(t, err)
		require.Equal(t, [32]byte{}, got)
	})

	t.Run("null value", func(t *testing.T) {
		t.Parallel()
		got, err := parseValueWord(json.RawMessage(`{"value":null}`))
		require.NoError(t, err)
		require.Equal(t, [32]byte{}, got)
	})

	t.Run("empty string value", func(t *testing.T) {
		t.Parallel()
		got, err := parseValueWord(json.RawMessage(`{"value":""}`))
		require.NoError(t, err)
		require.Equal(t, [32]byte{}, got)
	})

	t.Run("64 hex without 0x", func(t *testing.T) {
		t.Parallel()
		hexDigits := strings.Repeat("3a", 32)
		raw := json.RawMessage(`{"value":"` + hexDigits + `"}`)
		got, err := parseValueWord(raw)
		require.NoError(t, err)
		want := common.HexToHash("0x" + hexDigits)
		require.Equal(t, [32]byte(want), got)
	})

	t.Run("0x prefix", func(t *testing.T) {
		t.Parallel()
		hexDigits := strings.Repeat("01", 32)
		raw := json.RawMessage(`{"value":"0x` + hexDigits + `"}`)
		got, err := parseValueWord(raw)
		require.NoError(t, err)
		want := common.HexToHash("0x" + hexDigits)
		require.Equal(t, [32]byte(want), got)
	})

	t.Run("0X prefix", func(t *testing.T) {
		t.Parallel()
		hexDigits := strings.Repeat("fe", 32)
		raw := json.RawMessage(`{"value":"0X` + hexDigits + `"}`)
		got, err := parseValueWord(raw)
		require.NoError(t, err)
		want := common.HexToHash("0x" + hexDigits)
		require.Equal(t, [32]byte(want), got)
	})

	t.Run("max uint256", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"value":"` + strings.Repeat("f", 64) + `"}`)
		got, err := parseValueWord(raw)
		require.NoError(t, err)
		var want [32]byte
		for i := range want {
			want[i] = 0xff
		}
		require.Equal(t, want, got)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		_, err := parseValueWord(json.RawMessage(`{`))
		require.ErrorContains(t, err, "unmarshal stellar additionalFields")
	})

	t.Run("wrong hex length", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"value":"` + strings.Repeat("0", 62) + `"}`)
		_, err := parseValueWord(raw)
		require.ErrorContains(t, err, "value must be 32-byte hex")
	})

	t.Run("invalid hex digit", func(t *testing.T) {
		t.Parallel()
		raw := json.RawMessage(`{"value":"` + strings.Repeat("g", 64) + `"}`)
		_, err := parseValueWord(raw)
		require.ErrorContains(t, err, "invalid value hex")
	})
}

func TestEncoder_HashOperation_InvalidAdditionalFields(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(stellarTestnetSelector, 1, false)
	md := types.ChainMetadata{
		MCMAddress: "0x" + strings.Repeat("11", stellarContractIDBytes),
	}
	op := types.Operation{
		Transaction: types.Transaction{
			To:               "0x" + strings.Repeat("22", stellarContractIDBytes),
			AdditionalFields: json.RawMessage(`{`),
		},
	}
	_, err := enc.HashOperation(0, md, op)
	require.ErrorContains(t, err, "HashOperation: additionalFields.value:")
	require.ErrorContains(t, err, "unmarshal stellar additionalFields")
}
