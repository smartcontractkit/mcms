package stellar

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/types"
)

func Test_parseContractID_Errors(t *testing.T) {
	t.Parallel()

	_, err := parseContractID("")
	require.ErrorContains(t, err, "empty contract id")

	_, err = parseContractID("   ")
	require.ErrorContains(t, err, "empty contract id")

	// Not valid 64-char hex → falls through to strkey decode.
	_, err = parseContractID("0xabcd")
	require.ErrorContains(t, err, "decode contract strkey")

	_, err = parseContractID("not-a-strkey-or-hex")
	require.ErrorContains(t, err, "decode contract strkey")
}

func Test_parseContractID_HexVariants(t *testing.T) {
	t.Parallel()
	hexStr := "0x" + strings.Repeat("42", 32)
	want := common.HexToHash(hexStr)

	gotLower, err := parseContractID(hexStr)
	require.NoError(t, err)
	require.Equal(t, want, common.Hash(gotLower))

	gotUpperPrefix, err := parseContractID("0X" + strings.Repeat("42", 32))
	require.NoError(t, err)
	require.Equal(t, want, common.Hash(gotUpperPrefix))

	gotTrimmed, err := parseContractID("  \t" + hexStr + " \n")
	require.NoError(t, err)
	require.Equal(t, want, common.Hash(gotTrimmed))
}

func Test_chainNetworkID_KnownNetwork(t *testing.T) {
	t.Parallel()
	got, err := chainNetworkID(stellarTestnetSelector)
	require.NoError(t, err)

	wantHex, err := chainsel.StellarChainIdFromSelector(uint64(stellarTestnetSelector))
	require.NoError(t, err)
	require.True(t, common.IsHexHash(wantHex))
	want := common.HexToHash(wantHex)
	require.Equal(t, want, got)
}

func Test_chainNetworkID_InvalidSelector(t *testing.T) {
	t.Parallel()
	_, err := chainNetworkID(types.ChainSelector(0))
	require.Error(t, err)
	require.ErrorContains(t, err, "stellar chain id for selector 0")
}
