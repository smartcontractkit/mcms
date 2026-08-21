package stellar

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"
)

var ErrUint40Overflow = errors.New("value exceeds uint40 (2^40-1)")
var ErrUnsupportedEncodingVersion = errors.New("unsupported Stellar MCMS encoding version")
var sorobanSymbol = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)

// InvokePayload is the canonical Transaction.Data representation for Stellar.
// ArgsXDR is the XDR encoding of the argument ScVals (without the function).
type InvokePayload struct {
	Function string
	Args     []xdr.ScVal
	ArgsXDR  []byte
}

func EncodeSorobanInvokePayload(function string, args []xdr.ScVal) ([]byte, error) {
	return bindings.EncodeSorobanInvokePayload(function, args)
}

func DecodeSorobanInvokePayload(data []byte) (InvokePayload, error) {
	function, args, err := bindings.DecodeSorobanInvokePayload(data)
	if err != nil {
		return InvokePayload{}, err
	}
	av := xdr.ScVec(args)
	ap := &av
	argsXDR, err := (xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &ap}).MarshalBinary()
	if err != nil {
		return InvokePayload{}, fmt.Errorf("encode invoke arguments: %w", err)
	}

	return InvokePayload{Function: function, Args: args, ArgsXDR: argsXDR}, nil
}

func appendU32(buf *[]byte, value uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], value)
	*buf = append(*buf, b[:]...)
}
func appendU64(buf *[]byte, value uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], value)
	*buf = append(*buf, b[:]...)
}
func appendBytes32(buf *[]byte, value [32]byte) { *buf = append(*buf, value[:]...) }
func appendSized(buf *[]byte, value []byte) {
	appendU32(buf, uint32(len(value)))
	*buf = append(*buf, value...)
}

// Legacy ABI helpers are kept unexported so older package tests and downstream
// code compiled against the package continue to build during the migration.
func appendWord32(buf *[]byte, word [32]byte)           { *buf = append(*buf, word[:]...) }
func appendUint256FromBytes(buf *[]byte, word [32]byte) { appendWord32(buf, word) }
func appendUint40(buf *[]byte, value uint64) error {
	if value >= uint40MaxExclusive {
		return ErrUint40Overflow
	}
	var w [32]byte
	binary.BigEndian.PutUint64(w[24:], value)
	appendWord32(buf, w)

	return nil
}
func appendBool(buf *[]byte, value bool) {
	var w [32]byte
	if value {
		w[31] = 1
	}
	appendWord32(buf, w)
}
func appendABIBytes(buf *[]byte, value []byte) {
	var w [32]byte
	binary.BigEndian.PutUint64(w[24:], uint64(len(value)))
	appendWord32(buf, w)
	*buf = append(*buf, value...)
	for len(*buf)%32 != 0 {
		*buf = append(*buf, 0)
	}
}

func HashStellarRootMetadata(domain, networkID, multisig [32]byte, preOpCount, postOpCount uint64, override bool, configVersion uint64, version uint32) (common.Hash, error) {
	if version != encodingVersion {
		return common.Hash{}, fmt.Errorf("%w: %d", ErrUnsupportedEncodingVersion, version)
	}
	if preOpCount >= uint40MaxExclusive || postOpCount >= uint40MaxExclusive {
		return common.Hash{}, ErrUint40Overflow
	}
	buf := make([]byte, 0, 125)
	appendBytes32(&buf, domain)
	appendU32(&buf, version)
	appendBytes32(&buf, networkID)
	appendBytes32(&buf, multisig)
	appendU64(&buf, preOpCount)
	appendU64(&buf, postOpCount)
	if override {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	appendU64(&buf, configVersion)

	return crypto.Keccak256Hash(buf), nil
}

func HashCurrentStellarOp(domain, networkID, multisig [32]byte, nonce uint64, target [32]byte, function string, argsXDR []byte, version uint32) (common.Hash, error) {
	if version != encodingVersion {
		return common.Hash{}, fmt.Errorf("%w: %d", ErrUnsupportedEncodingVersion, version)
	}
	if nonce >= uint40MaxExclusive {
		return common.Hash{}, ErrUint40Overflow
	}
	if !sorobanSymbol.MatchString(function) {
		return common.Hash{}, fmt.Errorf("invalid Soroban symbol %q", function)
	}
	functionXDR, err := scval.SymbolToScVal(function).MarshalBinary()
	if err != nil {
		return common.Hash{}, fmt.Errorf("encode function: %w", err)
	}
	buf := make([]byte, 0, 184)
	appendBytes32(&buf, domain)
	appendU32(&buf, version)
	appendBytes32(&buf, networkID)
	appendBytes32(&buf, multisig)
	appendU64(&buf, nonce)
	appendBytes32(&buf, target)
	appendSized(&buf, functionXDR)
	appendSized(&buf, argsXDR)

	return crypto.Keccak256Hash(buf), nil
}

// Compatibility helper for the pre-versioned API. New code should use HashStellarRootMetadata.
func HashRootMetadata(domain, chainID, multisig [32]byte, preOpCount, postOpCount uint64, override bool) (common.Hash, error) {
	var buf []byte
	appendWord32(&buf, domain)
	appendWord32(&buf, chainID)
	appendWord32(&buf, multisig)
	if err := appendUint40(&buf, preOpCount); err != nil {
		return common.Hash{}, err
	}
	if err := appendUint40(&buf, postOpCount); err != nil {
		return common.Hash{}, err
	}
	appendBool(&buf, override)

	return crypto.Keccak256Hash(buf), nil
}

// HashStellarOp remains source-compatible for old callers; current encoders use HashCurrentStellarOp.
func HashStellarOp(domain, chainID, multisig [32]byte, nonce uint64, to, value [32]byte, data []byte) (common.Hash, error) {
	if nonce >= uint40MaxExclusive {
		return common.Hash{}, ErrUint40Overflow
	}
	var buf []byte
	appendWord32(&buf, domain)
	appendWord32(&buf, chainID)
	appendWord32(&buf, multisig)
	if err := appendUint40(&buf, nonce); err != nil {
		return common.Hash{}, err
	}
	appendWord32(&buf, to)
	appendWord32(&buf, value)
	var off [32]byte
	binary.BigEndian.PutUint64(off[24:], stellarOpDataABIByteOffset)
	appendWord32(&buf, off)
	appendABIBytes(&buf, data)

	return crypto.Keccak256Hash(buf), nil
}
