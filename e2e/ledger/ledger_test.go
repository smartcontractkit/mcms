//go:build ledger && !e2e
// +build ledger,!e2e

package ledger

import (
	"log"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// This test uses real ledger connected device. Remember to connect, unlock it and open ethereum app.
func TestManualLedgerSigning(t *testing.T) { //nolint:paralleltest
	t.Log("Starting manual Ledger signing test...")

	// Step 1: Detect and connect to the Ledger device
	t.Log("Checking for connected Ledger devices...")
	ledgerHub, err := usbwallet.NewLedgerHub()
	require.NoError(t, err, "Failed to initialize Ledger Hub")

	wallets := ledgerHub.Wallets()
	require.NotEmpty(t, wallets, "No Ledger devices found. Please connect your Ledger and unlock it.")

	// Use the first available wallet
	wallet := wallets[0]
	t.Logf("Found Ledger device: %s\n", wallet.URL().Path)

	// Open the wallet
	t.Log("Opening Ledger wallet...")
	err = wallet.Open("")
	require.NoError(t, err, "Failed to open Ledger wallet")

	t.Log("Ledger wallet opened successfully.")

	// Define the derivation path
	derivationPath := accounts.DefaultBaseDerivationPath

	// Derive the account and close the wallet
	account, err := wallet.Derive(derivationPath, true)
	if err != nil {
		log.Fatalf("Failed to derive account: %v", err)
	}
	t.Logf("Derived account: %s\n", account.Address.Hex())
	accountPublicKey := account.Address.Hex()
	wallet.Close()

	// Step 2: Load a proposal from a fixture
	t.Log("Loading proposal from fixture...")
	file, err := testutils.ReadFixture("proposal-testing.json")
	require.NoError(t, err, "Failed to read fixture") // Check immediately after ReadFixture
	defer func(file *os.File) {
		if file != nil {
			err = file.Close()
			require.NoError(t, err, "Failed to close file")
		}
	}(file)
	require.NoError(t, err)

	proposal, err := mcms.NewProposal(file)
	require.NoError(t, err, "Failed to parse proposal")
	t.Log("Proposal loaded successfully.")

	// Step 3: Create a Signable instance
	t.Log("Creating Signable instance...")
	inspectors := map[types.ChainSelector]sdk.Inspector{} // Provide required inspectors
	signable, err := mcms.NewSignable(proposal, inspectors)
	require.NoError(t, err, "Failed to create Signable instance")
	t.Log("Signable instance created successfully.")

	// Step 4: Create a LedgerSigner
	t.Log("Creating LedgerSigner...")
	ledgerSigner := mcms.NewLedgerSigner(derivationPath)

	// Step 5: Sign the proposal
	t.Log("Signing the proposal...")
	signature, err := signable.SignAndAppend(ledgerSigner)
	require.NoError(t, err, "Failed to sign proposal with Ledger")
	t.Log("Proposal signed successfully.")
	t.Logf("Signature: R=%s, S=%s, V=%d\n", signature.R.Hex(), signature.S.Hex(), signature.V)

	// Step 6: Validate the signature
	t.Log("Validating the signature...")
	hash, err := proposal.SigningHash()
	require.NoError(t, err, "Failed to compute proposal hash")

	recoveredAddr, err := signature.Recover(hash)
	require.NoError(t, err, "Failed to recover signer address")

	require.Equal(t, accountPublicKey, recoveredAddr.Hex(), "Signature verification failed")
	t.Logf("Signature verified successfully. Signed by: %s\n", recoveredAddr.Hex())
}
