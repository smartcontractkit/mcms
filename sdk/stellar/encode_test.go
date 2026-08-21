package stellar

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestDomainConstantsMatchKeccak256Literals(t *testing.T) {
	t.Parallel()
	require.Equal(t,
		crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_STELLAR")),
		common.Hash(domainOpStellar))
	require.Equal(t,
		crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_STELLAR")),
		common.Hash(domainMetaStellar))
}

// TestHashSetRootInnerGoldenVector validates the hash_set_root_inner preimage
// construction against a known golden value from the contract encoding tests.
func TestHashSetRootInnerGoldenVector(t *testing.T) {
	t.Parallel()
	root := [32]byte{}
	validUntil := uint32(0)
	var buf []byte
	appendBytes32(&buf, root)
	var vu [32]byte
	binary.BigEndian.PutUint32(vu[28:], validUntil)
	appendBytes32(&buf, vu)
	want := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
	require.Equal(t, want, crypto.Keccak256Hash(buf))
}

func hashBytes(t *testing.T, hexNoPrefix string) [32]byte {
	t.Helper()
	h := common.HexToHash("0x" + hexNoPrefix)

	return h
}
