package mcms

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

const SignatureBytesLength = 65
const EthereumSignatureVOffset = 27
const EthereumSignatureVThreshold = 2
const EthereumSignatureComponentSize = 32

type Signature struct {
	R common.Hash
	S common.Hash
	V uint8
}

func NewSignatureFromBytes(sig []byte) (Signature, error) {
	if len(sig) != SignatureBytesLength {
		return Signature{}, fmt.Errorf("invalid signature length: %d", len(sig))
	}

	return Signature{
		R: common.BytesToHash(sig[:EthereumSignatureComponentSize]),
		S: common.BytesToHash(sig[EthereumSignatureComponentSize:(SignatureBytesLength - 1)]),
		V: sig[SignatureBytesLength-1],
	}, nil
}

func (s Signature) ToGethSignature() gethwrappers.ManyChainMultiSigSignature {
	if s.V < EthereumSignatureVThreshold {
		s.V += EthereumSignatureVOffset
	}

	return gethwrappers.ManyChainMultiSigSignature{
		R: [32]byte(s.R.Bytes()),
		S: [32]byte(s.S.Bytes()),
		V: s.V,
	}
}

func (s Signature) ToBytes() []byte {
	return append(s.R.Bytes(), append(s.S.Bytes(), []byte{s.V}...)...)
}

func (s Signature) Recover(hash common.Hash) (common.Address, error) {
	return recoverAddressFromSignature(hash, s.ToBytes())
}

func recoverAddressFromSignature(hash common.Hash, sig []byte) (common.Address, error) {
	// The signature should be 65 bytes, and the last byte is the recovery id (v).
	if len(sig) != SignatureBytesLength {
		return common.Address{}, fmt.Errorf("invalid signature length")
	}

	// Adjust the recovery id (v) if needed. Ethereum signatures expect 27 or 28.
	// But `crypto.SigToPub` expects 0 or 1.
	if sig[SignatureBytesLength-1] > 1 {
		sig[SignatureBytesLength-1] -= EthereumSignatureVOffset
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
