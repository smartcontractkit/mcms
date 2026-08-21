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

	metaHashManual, err := HashStellarRootMetadata(
		domainMetaStellar,
		chainNet,
		hashBytes(t, metaAddr),
		0,
		1,
		false,
		1,
		encodingVersion,
	)
	require.NoError(t, err)

	md := types.ChainMetadata{
		StartingOpCount:  0,
		MCMAddress:       "0x" + metaAddr,
		AdditionalFields: json.RawMessage(`{"configVersion":1,"encodingVersion":1}`),
	}
	metaHashEnc, err := enc.HashMetadata(md)
	require.NoError(t, err)
	require.Equal(t, metaHashManual, metaHashEnc)

	tx, err := NewTransaction("0x"+toAddr, "accept_ownership", nil, "Ownable", nil)
	require.NoError(t, err)
	op := types.Operation{
		ChainSelector: stellarTestnetSelector,
		Transaction:   tx,
	}
	payload, err := DecodeSorobanInvokePayload(tx.Data)
	require.NoError(t, err)
	opHashManual, err := HashCurrentStellarOp(
		domainOpStellar,
		chainNet,
		hashBytes(t, metaAddr),
		0,
		hashBytes(t, toAddr),
		payload.Function,
		payload.ArgsXDR,
		encodingVersion,
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

func TestEncoder_HashOperation_InvalidAdditionalFields(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(stellarTestnetSelector, 1, false)
	payload, err := EncodeSorobanInvokePayload("accept_ownership", nil)
	require.NoError(t, err)
	md := types.ChainMetadata{
		MCMAddress: "0x" + strings.Repeat("11", stellarContractIDBytes),
	}
	op := types.Operation{
		Transaction: types.Transaction{
			To:               "0x" + strings.Repeat("22", stellarContractIDBytes),
			Data:             payload,
			AdditionalFields: json.RawMessage(`{`),
		},
	}
	_, err = enc.HashOperation(0, md, op)
	require.ErrorContains(t, err, "HashOperation: additional fields:")
	require.ErrorContains(t, err, "decode Stellar transaction additional fields")
}

func TestEncoder_HashOperation_RejectsUnsupportedOrNonCanonicalAdditionalFields(t *testing.T) {
	t.Parallel()
	enc := NewEncoder(stellarTestnetSelector, 1, false)
	md := types.ChainMetadata{MCMAddress: "0x" + strings.Repeat("11", stellarContractIDBytes)}
	payload, err := EncodeSorobanInvokePayload("accept_ownership", nil)
	require.NoError(t, err)

	for name, additionalFields := range map[string]json.RawMessage{
		"missing":             nil,
		"unsupported version": json.RawMessage(`{"family":"stellar","encodingVersion":2}`),
		"wrong family":        json.RawMessage(`{"family":"evm","encodingVersion":1}`),
		"unknown field":       json.RawMessage(`{"family":"stellar","encodingVersion":1,"target":"forbidden"}`),
	} {
		t.Run(name, func(t *testing.T) {
			op := types.Operation{Transaction: types.Transaction{
				To:               "0x" + strings.Repeat("22", stellarContractIDBytes),
				Data:             payload,
				AdditionalFields: additionalFields,
			}}
			_, err := enc.HashOperation(0, md, op)
			require.Error(t, err)
		})
	}
}
