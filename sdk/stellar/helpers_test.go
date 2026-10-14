package stellar_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func testMCMSConfig(t *testing.T) types.Config {
	t.Helper()

	cfg, err := types.NewConfig(1, []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")}, nil)
	require.NoError(t, err)

	return cfg
}

func testContractID(t *testing.T, seed byte) string {
	t.Helper()

	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = seed + byte(i)
	}

	id, err := strkey.Encode(strkey.VersionByteContract, raw)
	require.NoError(t, err)

	return id
}
