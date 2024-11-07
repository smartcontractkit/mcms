package mcms

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/types"
)

// Sign signs the proposal using the provided signer.
func Sign(signable *Signable, signer Signer) error {
	// Validate proposal
	if err := signable.Validate(); err != nil {
		return err
	}

	// Get the signing hash
	payload, err := signable.SigningHash()
	if err != nil {
		return err
	}

	// Sign the payload
	sigB, err := signer.Sign(payload.Bytes())
	if err != nil {
		return err
	}

	sig, err := types.NewSignatureFromBytes(sigB)
	if err != nil {
		return err
	}

	// Add signature to proposal
	signable.AddSignature(sig)

	return nil
}

// Signer is an interface for different strategies for signing payloads.
type Signer interface {
	Sign(payload []byte) ([]byte, error)
}

var _ Signer = &PrivateKeySigner{}

// PrivateKeySigner signs payloads using a private key.
type PrivateKeySigner struct {
	pk *ecdsa.PrivateKey
}

// NewPrivateKeySigner creates a new PrivateKeySigner.
func NewPrivateKeySigner(pk *ecdsa.PrivateKey) *PrivateKeySigner {
	return &PrivateKeySigner{pk: pk}
}

// Sign signs the payload using the private key.
func (s *PrivateKeySigner) Sign(payload []byte) ([]byte, error) {
	return crypto.Sign(payload, s.pk)
}

var _ Signer = &LedgerSigner{}

// LedgerSigner signs payloads using a Ledger.
type LedgerSigner struct {
	derivationPath []uint32
}

// NewLedgerSigner creates a new LedgerSigner.
func NewLedgerSigner(derivationPath []uint32) *LedgerSigner {
	return &LedgerSigner{derivationPath: derivationPath}
}

// Sign signs the payload using the first wallet found on a Ledger.
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
	return wallet.SignText(account, payload)
}
