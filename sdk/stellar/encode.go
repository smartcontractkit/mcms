package stellar

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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
	if len(value) > math.MaxUint32 {
		panic("value too large for uint32")
	}
	appendU32(buf, uint32(len(value))) //nolint:gosec // len checked above
	*buf = append(*buf, value...)
}

func HashStellarRootMetadata(domain, networkID, multisig [32]byte, preOpCount, postOpCount uint64, override bool, configVersion uint64, version uint32) (common.Hash, error) {
	if version != encodingVersion {
		return common.Hash{}, fmt.Errorf("%w: %d", ErrUnsupportedEncodingVersion, version)
	}
	if preOpCount >= uint40MaxExclusive || postOpCount >= uint40MaxExclusive {
		return common.Hash{}, ErrUint40Overflow
	}
	// 125 = 4*32 (domain+networkID+multisig+reserved) + 4 (version) + 2*8 (counts) + 1 (override) + 8 (configVersion)
	buf := make([]byte, 0, 125) //nolint:mnd
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
	// 184 = 4*32 (domain+networkID+multisig+target) + 4 (version) + 8 (nonce) + 2*4 (sizes) + functionXDR + argsXDR
	buf := make([]byte, 0, 184) //nolint:mnd
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
