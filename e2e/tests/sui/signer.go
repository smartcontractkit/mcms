//go:build e2e

package sui

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/blake2b"

	"github.com/smartcontractkit/chainlink-sui/bindings/utils"
)

type TestPrivateKeySigner struct {
	privateKey ed25519.PrivateKey
}

// NewTestPrivateKeySigner creates a new test signer from a private key
func NewTestPrivateKeySigner(privateKey ed25519.PrivateKey) utils.SuiSigner {
	return &TestPrivateKeySigner{
		privateKey: privateKey,
	}
}

// Sign implements SuiSigner
func (s *TestPrivateKeySigner) Sign(message []byte) ([]string, error) {
	// Add intent scope for transaction data (0x00, 0x00, 0x00)
	intentMessage := append([]byte{0x00, 0x00, 0x00}, message...)

	// Hash the message with blake2b
	hash := blake2b.Sum256(intentMessage)

	// Sign the hash
	signature := ed25519.Sign(s.privateKey, hash[:])

	// Get public key
	publicKey := s.privateKey.Public().(ed25519.PublicKey)

	// Create serialized signature: flag + signature + pubkey
	serializedSig := make([]byte, 1+len(signature)+len(publicKey))
	serializedSig[0] = 0x00 // Ed25519 flag
	copy(serializedSig[1:], signature)
	copy(serializedSig[1+len(signature):], publicKey)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(serializedSig)

	return []string{encoded}, nil
}

// GetAddress implements SuiSigner
func (s *TestPrivateKeySigner) GetAddress() (string, error) {
	publicKey := s.privateKey.Public().(ed25519.PublicKey)

	// For Ed25519, the signature scheme is 0x00
	const signatureScheme = 0x00

	// Create the data to hash: signature scheme byte || public key
	data := append([]byte{signatureScheme}, publicKey...)

	// Hash using Blake2b-256
	hash := blake2b.Sum256(data)

	// The Sui address is the hex representation of the hash
	return "0x" + hex.EncodeToString(hash[:]), nil
}
