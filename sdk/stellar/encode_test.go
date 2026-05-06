package stellar

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/utils/abi"
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

func TestHashRootMetadataMatchesABIEncoder(t *testing.T) {
	t.Parallel()
	domain := domainMetaStellar
	chainID := hashBytes(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	multisig := hashBytes(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	pre := uint64(7)
	post := uint64(9)
	override := true

	got, err := HashRootMetadata(domain, chainID, multisig, pre, post, override)
	require.NoError(t, err)

	metaABI := `[{"type":"bytes32"},{"type":"tuple","components":[
{"name":"chainId","type":"uint256"},
{"name":"multisig","type":"uint256"},
{"name":"preOpCount","type":"uint40"},
{"name":"postOpCount","type":"uint40"},
{"name":"overridePreviousRoot","type":"bool"}
]}]`
	metaTuple := struct {
		ChainID              *big.Int `abi:"chainId"`
		Multisig             *big.Int `abi:"multisig"`
		PreOpCount           *big.Int `abi:"preOpCount"`
		PostOpCount          *big.Int `abi:"postOpCount"`
		OverridePreviousRoot bool     `abi:"overridePreviousRoot"`
	}{
		ChainID:              new(big.Int).SetBytes(chainID[:]),
		Multisig:             new(big.Int).SetBytes(multisig[:]),
		PreOpCount:           big.NewInt(int64(pre)),
		PostOpCount:          big.NewInt(int64(post)),
		OverridePreviousRoot: override,
	}
	encoded, err := abi.Encode(metaABI, common.Hash(domain), metaTuple)
	require.NoError(t, err)
	want := crypto.Keccak256Hash(encoded)
	require.Equal(t, want, got)
}

func TestHashStellarOpGoldenVector(t *testing.T) {
	t.Parallel()
	domain := domainOpStellar
	chainID := hashBytes(t, "1111111111111111111111111111111111111111111111111111111111111111")
	multisig := hashBytes(t, "2222222222222222222222222222222222222222222222222222222222222222")
	nonce := uint64(42)
	to := hashBytes(t, "3333333333333333333333333333333333333333333333333333333333333333")
	value := hashBytes(t, "4444444444444444444444444444444444444444444444444444444444444444")
	data := []byte{1, 2, 3, 4, 5, 6, 7}

	got, err := HashStellarOp(domain, chainID, multisig, nonce, to, value, data)
	require.NoError(t, err)

	// Golden: keccak256 preimage must match Soroban contracts/mcms/src/abi_encoding.rs (layout is
	// domain || head fields || offset 192 || abi.bytes(data); not Solidity abi.encode(bytes32,tuple),
	// which inserts an extra dynamic offset after domain).
	want := common.HexToHash("0x6b0c3185f2fdaa391319dd36722b18b6d8d7566c4afaf70aaff50e18557f126b")
	require.Equal(t, want, got)

	var buf []byte
	appendWord32(&buf, domain)
	appendUint256FromBytes(&buf, chainID)
	appendUint256FromBytes(&buf, multisig)
	require.NoError(t, appendUint40(&buf, nonce))
	appendUint256FromBytes(&buf, to)
	appendUint256FromBytes(&buf, value)
	var off [32]byte
	binary.BigEndian.PutUint64(off[abiWordBytes-uint64ByteLen:], stellarOpDataABIByteOffset)
	appendWord32(&buf, off)
	appendABIBytes(&buf, data)
	require.Equal(t, want, crypto.Keccak256Hash(buf), "manual preimage matches HashStellarOp")
}

func TestHashRootMetadataUint40Overflow(t *testing.T) {
	t.Parallel()
	bad := uint40MaxExclusive
	_, err := HashRootMetadata(domainMetaStellar, [32]byte{}, [32]byte{}, 0, bad, false)
	require.ErrorIs(t, err, ErrUint40Overflow)
}

func TestHashStellarOpUint40Overflow(t *testing.T) {
	t.Parallel()
	_, err := HashStellarOp(domainOpStellar, [32]byte{}, [32]byte{}, uint40MaxExclusive, [32]byte{}, [32]byte{}, nil)
	require.ErrorIs(t, err, ErrUint40Overflow)
}

// Golden vectors from chainlink-stellar contracts/mcms/src/abi_encoding.rs tests.
func TestHashSetRootInnerGoldenVector(t *testing.T) {
	t.Parallel()
	root := [32]byte{}
	var buf []byte
	appendWord32(&buf, root)
	var vu [32]byte
	binary.BigEndian.PutUint32(vu[abiWordBytes-uint32ByteLen:], 0)
	appendWord32(&buf, vu)
	want := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
	require.Equal(t, want, crypto.Keccak256Hash(buf))
}

func hashBytes(t *testing.T, hexNoPrefix string) [32]byte {
	t.Helper()
	h := common.HexToHash("0x" + hexNoPrefix)

	return h
}

func Test_appendABIBytes_empty(t *testing.T) {
	t.Parallel()
	var buf []byte
	appendABIBytes(&buf, nil)
	require.Len(t, buf, abiWordBytes)
	var zeroWord [32]byte
	require.Equal(t, zeroWord[:], buf, "length word is zero; no payload or padding for empty bytes")
}
