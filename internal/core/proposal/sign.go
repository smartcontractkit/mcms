package proposal

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/smartcontractkit/mcms/sdk"

	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/types"
)

// Just run this locally to sign from the ledger.
func SignPlainKey(
	privateKey *ecdsa.PrivateKey,
	proposal Proposal,
	isSim bool,
	inspectors map[types.ChainSelector]sdk.Inspector,
) error {
	// Validate proposal
	err := proposal.Validate()
	if err != nil {
		return err
	}

	signable, err := proposal.Signable(isSim, inspectors)
	if err != nil {
		return err
	}

	// Get the signing hash
	payload, err := signable.SigningHash()
	if err != nil {
		return err
	}

	// Sign the payload
	sig, err := crypto.Sign(payload.Bytes(), privateKey)
	if err != nil {
		return err
	}

	// Unmarshal signature
	sigObj, err := types.NewSignatureFromBytes(sig)
	if err != nil {
		return err
	}

	// Add signature to proposal
	proposal.AddSignature(sigObj)

	return nil
}

func SignLedger(
	derivationPath []uint32,
	proposal Proposal,
	isSim bool,
	inspectors map[types.ChainSelector]sdk.Inspector,
) error {
	// Validate proposal
	if err := proposal.Validate(); err != nil {
		return fmt.Errorf("failed to validate proposal: %w", err)
	}

	// Load ledger
	ledgerhub, err := usbwallet.NewLedgerHub()
	if err != nil {
		return fmt.Errorf("failed to open ledger hub: %w", err)
	}

	// Get the first wallet
	wallets := ledgerhub.Wallets()
	if len(wallets) == 0 {
		return errors.New("no wallets found")
	}
	wallet := wallets[0]

	// Open the ledger.
	if err = wallet.Open(""); err != nil {
		return fmt.Errorf("failed to open wallet: %w", err)
	}
	defer wallet.Close()

	// Load account.
	account, err := wallet.Derive(derivationPath, true)
	if err != nil {
		return fmt.Errorf("is your ledger ethereum app open? Failed to derive account: %w derivation path %v", err, derivationPath)
	}

	// Create signable
	signable, err := proposal.Signable(isSim, inspectors)
	if err != nil {
		return err
	}

	payload, err := signable.SigningHash()
	if err != nil {
		return err
	}

	// Sign the payload with EIP 191.
	sig, err := wallet.SignText(account, payload.Bytes())
	if err != nil {
		return err
	}

	// Unmarshal signature
	sigObj, err := types.NewSignatureFromBytes(sig)
	if err != nil {
		return err
	}

	// Add signature to proposal
	proposal.AddSignature(sigObj)

	return nil
}
