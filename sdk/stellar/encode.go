package stellar

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrUint40Overflow = errors.New("value exceeds uint40 (2^40-1)")

func appendWord32(buf *[]byte, word [32]byte) {
	*buf = append(*buf, word[:]...)
}

func appendUint256FromBytes(buf *[]byte, word [32]byte) {
	appendWord32(buf, word)
}

func appendUint40(buf *[]byte, v uint64) error {
	if v >= uint40MaxExclusive {
		return fmt.Errorf("%w: %d", ErrUint40Overflow, v)
	}
	var w [32]byte
	var be [uint64ByteLen]byte
	binary.BigEndian.PutUint64(be[:], v)
	copy(w[abiWordBytes-uint40TailBytes:], be[uint64ByteLen-uint40TailBytes:])
	appendWord32(buf, w)

	return nil
}

func appendBool(buf *[]byte, v bool) {
	var w [32]byte
	if v {
		w[abiWordBytes-1] = 1
	}
	appendWord32(buf, w)
}

// appendABIBytes implements Solidity ABI encoding for `bytes`: length word + payload + right pad.
func appendABIBytes(buf *[]byte, data []byte) {
	n := uint64(len(data))
	var lenWord [32]byte
	binary.BigEndian.PutUint64(lenWord[abiWordBytes-uint64ByteLen:], n)
	appendWord32(buf, lenWord)
	*buf = append(*buf, data...)
	pad := (abiWordBytes - (len(data) % abiWordBytes)) % abiWordBytes
	for range pad {
		*buf = append(*buf, 0)
	}
}

// HashRootMetadata is keccak256(abi.encode(domain, StellarRootMetadata)) matching
// contracts/mcms/src/abi_encoding.rs hash_root_metadata.
func HashRootMetadata(
	domain [32]byte,
	chainID [32]byte,
	multisig [32]byte,
	preOpCount, postOpCount uint64,
	overridePreviousRoot bool,
) (common.Hash, error) {
	var buf []byte
	appendWord32(&buf, domain)
	appendUint256FromBytes(&buf, chainID)
	appendUint256FromBytes(&buf, multisig)
	if err := appendUint40(&buf, preOpCount); err != nil {
		return common.Hash{}, err
	}
	if err := appendUint40(&buf, postOpCount); err != nil {
		return common.Hash{}, err
	}
	appendBool(&buf, overridePreviousRoot)

	return crypto.Keccak256Hash(buf), nil
}

// HashStellarOp is keccak256(abi.encode(domain, StellarOp)) matching
// contracts/mcms/src/abi_encoding.rs hash_stellar_op.
func HashStellarOp(
	domain [32]byte,
	chainID [32]byte,
	multisig [32]byte,
	nonce uint64,
	to [32]byte,
	value [32]byte,
	data []byte,
) (common.Hash, error) {
	var buf []byte
	appendWord32(&buf, domain)
	appendUint256FromBytes(&buf, chainID)
	appendUint256FromBytes(&buf, multisig)
	if err := appendUint40(&buf, nonce); err != nil {
		return common.Hash{}, err
	}
	appendUint256FromBytes(&buf, to)
	appendUint256FromBytes(&buf, value)
	// offset of dynamic `data` from start of inner tuple = 6 * 32 = 192
	var off [32]byte
	binary.BigEndian.PutUint64(off[abiWordBytes-uint64ByteLen:], stellarOpDataABIByteOffset)
	appendWord32(&buf, off)
	appendABIBytes(&buf, data)

	return crypto.Keccak256Hash(buf), nil
}
