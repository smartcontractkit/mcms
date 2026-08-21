package stellar

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCurrentEncodingMatchesContractGoldenVector(t *testing.T) {
	network := mustBytes32(t, "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	multisig := mustBytes32(t, "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f")
	target := mustBytes32(t, "404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f")
	args := mustHex(t, "000000100000000100000000")
	meta, err := HashStellarRootMetadata(domainMetaStellar, network, multisig, 0, 1, false, 1, 1)
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x66853c15b75d9d4083efb5d1860d2066d33b34e82483b7550593e876b671c478"), meta)
	op, err := HashCurrentStellarOp(domainOpStellar, network, multisig, 0, target, "schedule_batch", args, 1)
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x070f68c72db975fda1c47c4a1d5cf2c7b49e92da406140f53c5276def4094ad3"), op)
}

func mustBytes32(t *testing.T, s string) [32]byte {
	t.Helper()
	var out [32]byte
	copy(out[:], mustHex(t, s))

	return out
}
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)

	return b
}
