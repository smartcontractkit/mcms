package types

import (
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// SignatureBytesLength defines the length of the signature in bytes after summing the byte
	// values of R, S, and V.
	SignatureBytesLength = 65

	// SignatureComponentSize defines the size of each signature component (R and S) in bytes.
	SignatureComponentSize = 32

	// SignatureVOffset defines the offset to adjust the recovery id (v) if needed.
	SignatureVOffset = 27
)

// Signature represents an signature that has been signed by a private key.
type Signature struct {
	R common.Hash
	S common.Hash
	V uint8
}

// NewSignatureFromBytes creates a new Signature from a byte slice of concatenated R, S, and V
// values.
func NewSignatureFromBytes(sig []byte) (Signature, error) {
	if len(sig) != SignatureBytesLength {
		return Signature{}, fmt.Errorf("invalid signature length: %d", len(sig))
	}

	return Signature{
		R: common.BytesToHash(sig[:SignatureComponentSize]),
		S: common.BytesToHash(sig[SignatureComponentSize:(SignatureBytesLength - 1)]),
		V: sig[SignatureBytesLength-1],
	}, nil
}

// ToBytes returns the byte representation of the signature.
func (s Signature) ToBytes() []byte {
	return slices.Concat(
		s.R.Bytes(),
		s.S.Bytes(),
		[]byte{s.V},
	)
}

// Recover returns the address of the hash that been recovered from the signature.
func (s Signature) Recover(hash common.Hash) (common.Address, error) {
	sig := s.ToBytes()

	// The signature should be 65 bytes, and the last byte is the recovery id (v).
	if len(sig) != SignatureBytesLength {
		return common.Address{}, fmt.Errorf("invalid signature length")
	}

	// Adjust the recovery id (v) if needed. Ethereum signatures expect 27 or 28.
	// But `crypto.SigToPub` expects 0 or 1.
	if sig[SignatureBytesLength-1] > 1 {
		sig[SignatureBytesLength-1] -= SignatureVOffset
	}

	// Recover the public key from the signature and the message hash
	pubKey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	// Derive the Ethereum address from the public key
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	return recoveredAddr, nil
}
