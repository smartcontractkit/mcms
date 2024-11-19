package evm

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

// derivePrivateKeyFromMnemonic derives the private key and address from the mnemonic
func derivePrivateKeyFromMnemonic(mnemonic string, accountIndex uint32) (*ecdsa.PrivateKey, common.Address, error) {
	// Create a new wallet from the mnemonic
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to create wallet from mnemonic: %w", err)
	}

	// Define the derivation path for the account
	derivationPath := fmt.Sprintf("m/44'/60'/0'/0/%d", accountIndex)
	path := hdwallet.MustParseDerivationPath(derivationPath)

	// Derive the account
	account, err := wallet.Derive(path, false)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to derive account: %w", err)
	}

	// Get the private key for the account
	privateKey, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to get private key: %w", err)
	}

	return privateKey, account.Address, nil
}
