package mcms

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Applies the EIP191 prefix to the payload and hashes it.
func toEthSignedMessageHash(payload []byte) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, payload...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}
