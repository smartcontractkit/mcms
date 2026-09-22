//go:build e2e

package ledger

import (
	"context"
	"log"
	"math/big"
	"os"
	"testing"

	stellardeployer "github.com/smartcontractkit/chainlink-stellar/deployment"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/suite"

	chainsel "github.com/smartcontractkit/chain-selectors"

	stellarmcms "github.com/smartcontractkit/chainlink-stellar/deployment/mcmsutil"

	"github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	solanae2e "github.com/smartcontractkit/mcms/e2e/tests/solana"
	"github.com/smartcontractkit/mcms/e2e/tests/stellar"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	solanamcms "github.com/smartcontractkit/mcms/sdk/solana"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"

	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/gagliardetto/solana-go"
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

	authEVM         *bind.TransactOpts
	authSolana      solana.PrivateKey
	stellarDeployer *stellardeployer.Deployer

	chainSelectorEVM     types.ChainSelector
	chainSelectorSolana  types.ChainSelector
	chainSelectorStellar types.ChainSelector

	e2e.TestSetup
}

func (s *ManualLedgerSigningTestSuite) deployMCMContractEVM(ctx context.Context) (common.Address, *bindings.ManyChainMultiSig) {
	// Set auth keys
	chainID, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Failed to parse private key")
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.authEVM = auth

	// Deploy MCMS contract
	s.Require().True(ok, "Failed to parse chain ID")
	mcmsAddress, tx, instance, err := bindings.DeployManyChainMultiSig(s.authEVM, s.ClientA)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(ctx, s.ClientA, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")
	s.Require().Equal(gethTypes.ReceiptStatusSuccessful, receipt.Status)

	return mcmsAddress, instance
}

func (s *ManualLedgerSigningTestSuite) initializeMCMSolana(ctx context.Context) (solana.PublicKey, string) {
	var MCMProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	solanae2e.InitializeMCMProgram(
		ctx,
		s.T(),
		s.SolanaClient,
		MCMProgramID,
		mcmInstanceSeed,
		uint64(s.chainSelectorSolana),
	)

	programID := solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])
	contractAddress := solanamcms.ContractAddress(programID, mcmInstanceSeed)

	return programID, contractAddress
}

// setRootEVM initializes and MCMS contract and calls set root on it
func (s *ManualLedgerSigningTestSuite) setRootEVM(
	ctx context.Context,
	ledgerAccount common.Address,
	proposal *mcms.Proposal,
	instance *bindings.ManyChainMultiSig,
	executorsMap map[types.ChainSelector]sdk.Executor,
) {
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{ledgerAccount}}

	// set config
	configurer := evm.NewConfigurer(s.ClientA, s.authEVM)
	tx, err := configurer.SetConfig(ctx, instance.Address().Hex(), &mcmConfig, true)
	s.Require().NoError(err, "Failed to set contract configuration")
	_, err = bind.WaitMined(ctx, s.ClientA, tx.RawData.(*gethTypes.Transaction))
	s.Require().NoError(err, "Failed to mine set config transaction")

	// set root

	executable, err := mcms.NewExecutable(proposal, executorsMap) //nolint:contextcheck // OPT-400
	s.Require().NoError(err)
	tx, err = executable.SetRoot(ctx, s.chainSelectorEVM)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)
}

const privateKeyLedger = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

var mcmInstanceSeed = [32]byte{'t', 'e', 's', 't', '-', 's', 'e', 't', 'r', 'o', 'o', 't', 'l', 'e', 'd', 'g', 'e', 'r'}

// setRootSolana initializes and MCMS contract and calls set root on it
func (s *ManualLedgerSigningTestSuite) setRootSolana(
	ctx context.Context,
	mcmProgramID solana.PublicKey,
	ledgerAccount common.Address,
	proposal *mcms.Proposal,
	executorsMap map[types.ChainSelector]sdk.Executor,
) {
	// set config
	mcmAddress := solanamcms.ContractAddress(mcmProgramID, mcmInstanceSeed)
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{ledgerAccount}}
	configurer := solanamcms.NewConfigurer(s.SolanaClient, s.authSolana, s.chainSelectorSolana)
	_, err := configurer.SetConfig(ctx, mcmAddress, &mcmConfig, true)
	s.Require().NoError(err)

	// set root

	executable, err := mcms.NewExecutable(proposal, executorsMap) //nolint:contextcheck // OPT-400
	s.Require().NoError(err)
	tx, err := executable.SetRoot(ctx, s.chainSelectorSolana)
	s.Require().NoError(err)

	// --- assert ---
	_, err = solana.SignatureFromBase58(tx.Hash)
	s.Require().NoError(err)
}

func (s *ManualLedgerSigningTestSuite) initializeMCMSStellar(
	ctx context.Context,
	ledgerAccount common.Address,
) xdr.AccountId {
	kp, err := keypair.Random()
	s.Require().NoError(err)

	stellar.FundStellarKey(
		s.T(),
		s.StellarClient.URL(),
		kp,
	)

	deployer := stellardeployer.NewDeployer(
		s.StellarClient,
		chainsel.STELLAR_LOCALNET.Passphrase,
		kp,
	)

	salt := stellarmcms.MCMSDeploySalt(
		chainsel.STELLAR_LOCALNET.Selector,
		"",
	)

	// The Ledger account is the MCMS secp256k1 signer.
	// It is intentionally an EVM common.Address, not the Stellar
	// transaction-signing account.
	mcmConfig := &types.Config{
		Quorum:  1,
		Signers: []common.Address{ledgerAccount},
	}

	// chain-selectors stores the Stellar network ID as the chain ID.
	networkIDHex, err := chainsel.StellarChainIdFromSelector(
		chainsel.STELLAR_LOCALNET.Selector,
	)
	s.Require().NoError(err)
	s.Require().True(
		common.IsHexHash(networkIDHex),
		"invalid Stellar network ID %q",
		networkIDHex,
	)

	chainNetworkID := common.HexToHash(networkIDHex)

	contractID, err := stellarmcms.DeployMCMS(
		ctx,
		deployer,
		kp.Address(),
		chainNetworkID,
		mcmConfig,
		"ledger_test",
		salt,
	)
	s.Require().NoError(err)

	s.stellarDeployer = deployer

	return xdr.MustAddress(contractID)
}

func (s *ManualLedgerSigningTestSuite) setRootStellar(
	ctx context.Context,
	mcmAddress string,
	ledgerAccount common.Address,
	proposal *mcms.Proposal,
	executorsMap map[types.ChainSelector]sdk.Executor,
) {
	// Exercise the Configurer too, matching EVM/Solana.
	mcmConfig := types.Config{
		Quorum:  1,
		Signers: []common.Address{ledgerAccount},
	}

	configurer := stellarsdk.NewConfigurer(
		s.stellarDeployer,
	)

	_, err := configurer.SetConfig(
		ctx,
		mcmAddress,
		&mcmConfig,
		true,
	)
	s.Require().NoError(
		err,
		"Failed to configure Stellar MCMS",
	)

	executable, err := mcms.NewExecutable( //nolint:contextcheck // OPT-400
		proposal,
		executorsMap,
	)
	s.Require().NoError(err)

	_, err = executable.SetRoot(
		ctx,
		s.chainSelectorStellar,
	)
	s.Require().NoError(
		err,
		"Failed to set Stellar MCMS root",
	)

	inspector := stellarsdk.NewInspectorFromInvoker(
		s.stellarDeployer,
	)

	root, validUntil, err := inspector.GetRoot(
		ctx,
		mcmAddress,
	)
	s.Require().NoError(err)

	s.Require().NotEqual(
		common.Hash{},
		root,
		"Stellar MCMS root should have been set",
	)

	s.Require().NotZero(
		validUntil,
		"Stellar MCMS root should have a validity deadline",
	)
}

// This test uses real ledger connected device. Remember to connect, unlock it and open ethereum app.
func (s *ManualLedgerSigningTestSuite) TestManualLedgerSigning() {
	s.T().Log("Starting manual Ledger signing test...")

	ctx := context.Background()
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	chainDetailsEVM, err := chainsel.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.BlockchainA.Out.Family)
	s.Require().NoError(err)
	chainDetailsSolana, err := chainsel.GetChainDetailsByChainIDAndFamily(s.SolanaChain.ChainID, s.SolanaChain.Out.Family)
	s.Require().NoError(err)

	s.chainSelectorEVM = types.ChainSelector(chainDetailsEVM.ChainSelector)
	s.chainSelectorSolana = types.ChainSelector(chainDetailsSolana.ChainSelector)

	// Step 1: Detect and connect to the Ledger device
	s.T().Log("Checking for connected Ledger devices...")
	ledgerHub, err := usbwallet.NewLedgerHub()
	s.Require().NoError(err, "Failed to initialize Ledger Hub")

	wallets := ledgerHub.Wallets()
	s.Require().NotEmpty(wallets, "No Ledger devices found. Please connect your Ledger and unlock it.")

	// Use the first available wallet
	wallet := wallets[0]
	s.T().Logf("Found Ledger device: %s\n", wallet.URL().Path)

	// Open the wallet
	s.T().Log("Opening Ledger wallet...")
	err = wallet.Open("")
	s.Require().NoError(err, "Failed to open Ledger wallet")

	s.T().Log("Ledger wallet opened successfully.")

	// Define the derivation path
	derivationPath := accounts.DefaultBaseDerivationPath

	// Derive the account and close the wallet
	account, err := wallet.Derive(derivationPath, true)
	if err != nil {
		log.Fatalf("Failed to derive account: %v", err)
	}
	s.T().Logf("Derived account: %s\n", account.Address.Hex())
	accountPublicKey := account.Address.Hex()
	wallet.Close()

	// Step 2: Deploy and initialize solana and EVM MCMs
	mcmsAddressEVM, mcmInstanceEVM := s.deployMCMContractEVM(ctx)
	mcmProgramID, contractIDSolana := s.initializeMCMSolana(ctx)
	mcmIDStellar := s.initializeMCMSStellar(ctx, account.Address)

	// Step 3: Load a proposal from a fixture
	s.T().Log("Loading proposal from fixture...")
	file, err := testutils.ReadFixture("proposal-testing.json")
	s.Require().NoError(err, "Failed to read fixture") // Check immediately after ReadFixture
	defer func(file *os.File) {
		s.Require().NotNil(file)
		err = file.Close()
		s.Require().NoError(err, "Failed to close file")
	}(file)
	s.Require().NoError(err)

	proposal, err := mcms.NewProposal(file)
	s.Require().NoError(err, "Failed to parse proposal")
	s.T().Log("Proposal loaded successfully.")
	proposal.ChainMetadata[s.chainSelectorEVM] = types.ChainMetadata{
		MCMAddress:      mcmsAddressEVM.String(),
		StartingOpCount: 0,
	}
	proposal.ChainMetadata[s.chainSelectorSolana] = types.ChainMetadata{
		MCMAddress:      contractIDSolana,
		StartingOpCount: 0,
	}
	proposal.ChainMetadata[s.chainSelectorStellar] = types.ChainMetadata{
		MCMAddress:      mcmIDStellar.Address(),
		StartingOpCount: 0,
	}

	// Step 4: Create a Signable instance
	s.T().Log("Creating Signable instance...")
	stellarInspector := stellarsdk.NewInspectorFromInvoker(s.stellarDeployer)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelectorEVM:     evm.NewInspector(s.ClientA),
		s.chainSelectorSolana:  solanamcms.NewInspector(s.SolanaClient),
		s.chainSelectorStellar: stellarInspector,
	}
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	authSolana, err := solana.PrivateKeyFromBase58(privateKeyLedger)
	s.Require().NoError(err)
	s.authSolana = authSolana
	encoderEVM := encoders[s.chainSelectorEVM].(*evm.Encoder)
	encoderSolana := encoders[s.chainSelectorSolana].(*solanamcms.Encoder)
	encoderStellar := encoders[s.chainSelectorStellar].(*stellarsdk.Encoder)
	executorEVM := evm.NewExecutor(encoderEVM, s.ClientA, s.authEVM)
	executorSolana := solanamcms.NewExecutor(encoderSolana, s.SolanaClient, authSolana)
	executorStellar, err := stellarsdk.NewExecutor(
		encoderStellar,
		stellarInspector,
	)
	s.Require().NoError(err)
	executorsMap := map[types.ChainSelector]sdk.Executor{
		s.chainSelectorEVM:     executorEVM,
		s.chainSelectorSolana:  executorSolana,
		s.chainSelectorStellar: executorStellar,
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err, "Failed to create Signable instance")
	s.T().Log("Signable instance created successfully.")

	// Step 5: Create a LedgerSigner and sign the proposal
	s.T().Log("Creating LedgerSigner...")
	ledgerSigner := mcms.NewLedgerSigner(derivationPath)

	s.T().Log("Signing the proposal...")
	signature, err := signable.SignAndAppend(ledgerSigner)
	s.Require().NoError(err, "Failed to sign proposal with Ledger")
	s.T().Log("Proposal signed successfully.")
	s.T().Logf("Signature: R=%s, S=%s, V=%d\n", signature.R.Hex(), signature.S.Hex(), signature.V)

	// Step 6: Validate the signature
	s.T().Log("Validating the signature...")
	hash, err := proposal.SigningHash()
	s.Require().NoError(err, "Failed to compute proposal hash")

	recoveredAddr, err := signature.Recover(hash)
	s.Require().NoError(err, "Failed to recover signer address")

	s.Require().Equal(accountPublicKey, recoveredAddr.Hex(), "Signature verification failed")
	s.T().Logf("Signature verified successfully. Signed by: %s\n", recoveredAddr.Hex())

	// Step 7: Call Set Root to verify signature
	s.setRootEVM(ctx, account.Address, proposal, mcmInstanceEVM, executorsMap)
	s.setRootSolana(ctx, mcmProgramID, account.Address, proposal, executorsMap)
	s.setRootStellar(ctx, mcmIDStellar.Address(), account.Address, proposal, executorsMap)
}
