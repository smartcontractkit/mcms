package mcms

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/sdk/usbwallet"
)

// signer is an interface for different strategies for signing payloads.
type signer interface {
	Sign(payload []byte) ([]byte, error)
	GetAddress() (common.Address, error)
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
	fmt.Println("[DEBUG]", "Signing payload with private key:", common.Bytes2Hex(payload))
	return crypto.Sign(toEthSignedMessageHash(payload).Bytes(), s.pk)
}

// GetAddress returns the address of the signer.
func (s *PrivateKeySigner) GetAddress() (common.Address, error) {
	return crypto.PubkeyToAddress(s.pk.PublicKey), nil
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
	wallet, account, err := s.setupLedgerAccount()
	if err != nil {
		return nil, err
	}
	defer wallet.Close()

	// Sign the payload with EIP 191
	return wallet.SignText(account, payload[:])
}

func (s *LedgerSigner) GetAddress() (common.Address, error) {
	wallet, account, err := s.setupLedgerAccount()
	if err != nil {
		return common.Address{}, err
	}
	defer wallet.Close()

	return account.Address, nil
}

// setupLedgerAccount loads the wallet and account from the ledger. Caller is responsible for closing the wallet.
func (s *LedgerSigner) setupLedgerAccount() (accounts.Wallet, accounts.Account, error) {
	// Load ledger
	ledgerhub, err := usbwallet.NewLedgerHub()
	if err != nil {
		return nil, accounts.Account{}, fmt.Errorf("failed to open ledger hub: %w", err)
	}

	// Get the first wallet
	wallets := ledgerhub.Wallets()
	if len(wallets) == 0 {
		return nil, accounts.Account{}, errors.New("no wallets found")
	}
	wallet := wallets[0]

	// Open the ledger
	if err = wallet.Open(""); err != nil {
		return nil, accounts.Account{}, fmt.Errorf("failed to open wallet: %w", err)
	}

	// Load account
	account, err := wallet.Derive(s.derivationPath, true)
	if err != nil {
		wallet.Close() // Only close on error since caller won't be able to
		return nil, accounts.Account{}, fmt.Errorf("is your ledger ethereum app open? Failed to derive account: %w derivation path %v", err, s.derivationPath)
	}

	return wallet, account, nil
}
