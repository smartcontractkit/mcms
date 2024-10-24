package mcms

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const SignatureBytesLength = 65

const EthereumSignatureComponentSize = 32
const EthereumSignatureVOffset = 27

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
