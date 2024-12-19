package mcms

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/sdk/usbwallet"
)

// signer is an interface for different strategies for signing payloads.
type signer interface {
	Sign(payload []byte) ([]byte, error)
}

var _ signer = &PrivateKeySigner{}

// PrivateKeySigner signs payloads using a private key.
type PrivateKeySigner struct {
	pk *ecdsa.PrivateKey
}

// NewPrivateKeySigner creates a new PrivateKeySigner.
func NewPrivateKeySigner(pk *ecdsa.PrivateKey) *PrivateKeySigner {
	return &PrivateKeySigner{pk: pk}
}

// Sign signs the payload using the private key.
// The payload here should be without the EIP 191 prefix,
// and the function will add it before signing.
func (s *PrivateKeySigner) Sign(payload []byte) ([]byte, error) {
	return crypto.Sign(toEthSignedMessageHash(payload).Bytes(), s.pk)
}

var _ signer = &LedgerSigner{}

// LedgerSigner signs payloads using a Ledger.
type LedgerSigner struct {
	derivationPath []uint32
}

// NewLedgerSigner creates a new LedgerSigner.
func NewLedgerSigner(derivationPath []uint32) *LedgerSigner {
	return &LedgerSigner{derivationPath: derivationPath}
}

// Sign signs the payload using the first wallet found on a Ledger.
// The payload here should be without the EIP 191 prefix,
// and the ledger will add it before signing.
func (s *LedgerSigner) Sign(payload []byte) ([]byte, error) {
	// Load ledger
	ledgerhub, err := usbwallet.NewLedgerHub()
	if err != nil {
		return nil, fmt.Errorf("failed to open ledger hub: %w", err)
	}

	// Get the first wallet
	wallets := ledgerhub.Wallets()
	if len(wallets) == 0 {
		return nil, errors.New("no wallets found")
	}
	wallet := wallets[0]

	// Open the ledger
	if err = wallet.Open(""); err != nil {
		return nil, fmt.Errorf("failed to open wallet: %w", err)
	}
	defer wallet.Close()

	// Load account
	account, err := wallet.Derive(s.derivationPath, true)
	if err != nil {
		return nil, fmt.Errorf("is your ledger ethereum app open? Failed to derive account: %w derivation path %v", err, s.derivationPath)
	}

	// Sign the payload with EIP 191
	return wallet.SignText(account, payload[:])
}
