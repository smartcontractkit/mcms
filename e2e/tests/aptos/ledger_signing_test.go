//go:build e2e && aptosledger

package aptos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/usbwallet"
	"github.com/smartcontractkit/mcms/types"
)

const (
	deployerKey = "ed25519-priv-0x1234"
)

var (
	chainSelector = types.ChainSelector(chain_selectors.APTOS_TESTNET.Selector)
)

func TestAptosLedgerSetup(t *testing.T) {
	// Signers in each group need to be sorted alphabetically
	config := &types.Config{
		Quorum: 1,
		Signers: []common.Address{
			common.HexToAddress("0x84d9CB2835DBF54Be56948fDf133d14A46859690"),
		},
		GroupSigners: nil,
	}
	suite := AptosTestSuite{
		ChainSelector: chainSelector,
	}
	suite.SetT(t)
	testnetClient, err := aptos.NewNodeClient("https://api.testnet.aptoslabs.com/v1", 0)
	require.NoError(t, err)
	suite.TestSetup.AptosRPCClient = testnetClient

	deployer := &crypto.Ed25519PrivateKey{}
	deployer.FromHex(deployerKey)
	suite.deployerAccount, _ = aptos.NewAccountFromSigner(deployer)

	suite.deployMCM()
	suite.deployMCMUser()

	configurer := aptossdk.NewConfigurer(suite.AptosRPCClient, suite.deployerAccount)
	_, err = configurer.SetConfig(context.Background(), suite.MCMContract.StringLong(), config, true)
	suite.Require().NoError(err, "setting config on Aptos mcms contract")

	// Build proposal
	validUntil := time.Now().Add(time.Hour).Unix()
	proposalBuilder := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil)).
		SetDescription("Test proposal with Ledger signing on Aptos").
		SetOverridePreviousRoot(true).
		AddChainMetadata(suite.ChainSelector, types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      suite.MCMContract.StringLong(),
		})

	// Call 1
	additionalFields := aptossdk.AdditionalFields{
		ModuleName: "mcms_user",
		Function:   "function_one",
	}
	callOneParamBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.WriteString("helloworld")
		ser.WriteBytes([]byte{5, 4, 3, 2, 1})
	})
	suite.Require().NoError(err)
	callOneAdditionalFields, _ := json.Marshal(additionalFields)
	proposalBuilder.AddOperation(types.Operation{
		ChainSelector: suite.ChainSelector,
		Transaction: types.Transaction{
			To:               suite.MCMSUserContract.StringLong(),
			Data:             callOneParamBytes,
			AdditionalFields: callOneAdditionalFields,
		},
	})

	// Call 2
	additionalFields = aptossdk.AdditionalFields{
		ModuleName: "mcms_user",
		Function:   "function_two",
	}
	callTwoParamBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.FixedBytes(suite.deployerAccount.Address[:])
		ser.U128(*big.NewInt(42))
	})
	suite.Require().NoError(err)
	callTwoAdditionalFields, _ := json.Marshal(additionalFields)
	proposalBuilder.AddOperation(types.Operation{
		ChainSelector: suite.ChainSelector,
		Transaction: types.Transaction{
			To:               suite.MCMSUserContract.StringLong(),
			Data:             callTwoParamBytes,
			AdditionalFields: callTwoAdditionalFields,
		},
	})

	proposal, err := proposalBuilder.Build()
	suite.Require().NoError(err, "Error building proposal")

	buff := &bytes.Buffer{}
	err = mcms.WriteProposal(buff, proposal)
	suite.Require().NoError(err, "Error writing proposal")

	fmt.Println(buff.String())
}

const proposal =
// language=json
`
{
  "version": "v1",
  "kind": "Proposal",
  ...
}
`

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
		t.Fatalf("Failed to derive account: %v", err)
	}
	t.Logf("Derived account: %s\n", account.Address.Hex())
	accountPublicKey := account.Address.Hex()
	wallet.Close()

	// Step 2: Load proposal
	t.Log("Loading proposal...")
	buff := bytes.NewBufferString(proposal)

	proposal, err := mcms.NewProposal(buff)
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

	buff = &bytes.Buffer{}
	mcms.WriteProposal(buff, proposal)
	t.Log("Signed the proposal successfully.")
	fmt.Println(buff.String())
}

func TestAptosSetRootExecute(t *testing.T) {
	testnetClient, err := aptos.NewNodeClient("https://api.testnet.aptoslabs.com/v1", 0)
	require.NoError(t, err)

	deployer := &crypto.Ed25519PrivateKey{}
	deployer.FromHex(deployerKey)
	deployerAccount, _ := aptos.NewAccountFromSigner(deployer)

	buff := bytes.NewBufferString(proposal)
	proposal, err := mcms.NewProposal(buff)
	require.NoError(t, err, "Failed to parse proposal")

	encoders, err := proposal.GetEncoders()
	require.NoError(t, err)
	aptosEncoder := encoders[chainSelector].(*aptossdk.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{
		chainSelector: aptossdk.NewExecutor(testnetClient, deployerAccount, aptosEncoder),
	}
	executable, err := mcms.NewExecutable(proposal, executors)
	require.NoError(t, err)

	// Set Root
	t.Log("Setting root...")
	txOutput, err := executable.SetRoot(context.Background(), chainSelector)
	require.NoError(t, err)

	t.Logf("✅ SetRoot in tx: %s", txOutput.Hash)

	// Execute Operations
	for i := range proposal.Operations {
		t.Logf("Executing operation: %v...", i)
		txOutput, err = executable.Execute(context.Background(), i)
		require.NoError(t, err)
		t.Logf("✅ Executed Operation in tx: %s", txOutput.Hash)
	}
}
