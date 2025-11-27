package testutils

import (
	"crypto/ecdsa"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Note: should only be used for testing purposes
type ECDSASigner struct {
	Key *ecdsa.PrivateKey
}

func NewECDSASigner() *ECDSASigner {
	key, _ := crypto.GenerateKey()
	return &ECDSASigner{Key: key}
}

func (s *ECDSASigner) Address() common.Address {
	return crypto.PubkeyToAddress(s.Key.PublicKey)
}

func MakeNewECDSASigners(n int) []ECDSASigner {
	signers := make([]ECDSASigner, n)
	for i := range n {
		signers[i] = *NewECDSASigner()
	}
	// Signers need to be sorted alphabetically
	slices.SortFunc(signers[:], func(a, b ECDSASigner) int {
		return strings.Compare(strings.ToLower(a.Address().Hex()), strings.ToLower(b.Address().Hex()))
	})
	return signers
}
