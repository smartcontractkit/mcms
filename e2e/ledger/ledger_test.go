//go:build e2e
// +build e2e

package ledger

import (
	"context"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func TestManualLedgerSigningSuite(t *testing.T) {
	var runLedgerSuite = os.Getenv("RUN_LEDGER_SUITE") == "true"
	if !runLedgerSuite {
		t.Skip("Skipping LedgerSuite. Set RUN_LEDGER_SUITE=true to run it.")
	}
	suite.Run(t, new(ManualLedgerSigningTestSuite))
}

// ManualLedgerSigningTestSuite tests the manual ledger signing functionality
type ManualLedgerSigningTestSuite struct {
	suite.Suite
	mcmsContractEVM     *bindings.ManyChainMultiSig
	deployerKey         common.Address
	auth                *bind.TransactOpts
	chainSelectorEVM    types.ChainSelector
	chainSelectorSolana types.ChainSelector
	e2e.TestSetup
}

// setRootEVM initializes and MCMS contract and calls set root on it
func (s *ManualLedgerSigningTestSuite) setRootEVM(ctx context.Context, ledgerAccount common.Address, proposal *mcms.Proposal) *bindings.ManyChainMultiSig {

	chainID, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")
	_, tx, instance, err := bindings.DeployManyChainMultiSig(s.auth, s.Client)
	s.Require().NoError(err, "Failed to deploy contract")

	// Set auth keys
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(ctx, s.Client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(gethTypes.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0}   // One groups: Group 0
	groupQuorums := [32]uint8{1} // Quorum 1 for group 0
	groupParents := [32]uint8{0} // Group 0 is its own parent
	signers := []common.Address{ledgerAccount}
	clearRoot := true

	tx, err = instance.SetConfig(s.auth, signers, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	receipt, err = bind.WaitMined(context.Background(), s.Client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
	s.Require().Equal(gethTypes.ReceiptStatusSuccessful, receipt.Status)

	// Set Root
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelectorEVM].(*evm.Encoder)

	executor := evm.NewExecutor(encoder, s.Client, s.auth)
	executorsMap := map[types.ChainSelector]sdk.Executor{
		s.chainSelectorEVM: executor,
	}
	executable, err := mcms.NewExecutable(proposal, executorsMap)
	s.Require().NoError(err)
	txHash, err := executable.SetRoot(ctx, s.chainSelectorEVM)
	s.Require().NoError(err)
	s.Require().NotEmpty(txHash)
	return instance
}

// This test uses real ledger connected device. Remember to connect, unlock it and open ethereum app.
func (s *ManualLedgerSigningTestSuite) TestManualLedgerSigning() {
	t := s.T()
	s.TestSetup = *e2e.InitializeSharedTestSetup(t)
	var runLedgerSuite = os.Getenv("RUN_LEDGER_SUITE") == "true"
	if !runLedgerSuite {
		s.T().Skip("Skipping LedgerSuite. Set RUN_LEDGER_SUITE=true to run it.")
	}
	ctx := context.Background()

	chainDetailsEVM, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.Config.BlockchainA.Out.Family)
	s.Require().NoError(err)
	chainDetailsSolana, err := cselectors.GetChainDetailsByChainIDAndFamily(s.SolanaChain.Out.ChainID, s.Config.SolanaChain.Out.Family)
	s.Require().NoError(err)

	s.chainSelectorEVM = types.ChainSelector(chainDetailsEVM.ChainSelector)
	s.chainSelectorSolana = types.ChainSelector(chainDetailsSolana.ChainSelector)
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
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelectorEVM:    evm.NewInspector(s.Client),
		s.chainSelectorSolana: solana.NewInspector(s.SolanaClient),
	}
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

	// Step 7: Call Set Root to verify signature
	s.setRootEVM(ctx, account.Address, proposal)
}
