//go:build e2e
// +build e2e

package evm

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

// Config defines the blockchain configuration
type Config struct {
	BlockchainA *blockchain.Input `toml:"evm_config" validate:"required"`
	Settings    struct {
		PrivateKey string `toml:"private_key" validate:"required"`
	} `toml:"settings"`
}

// TestContext holds shared resources for the test suite
type TestContext struct {
	Client          *ethclient.Client
	ContractAddress string
	DeployerKey     common.Address
	SignerAddresses []common.Address
}

// TestInspection serves as the setup suite
func TestInspection(t *testing.T) {
	t.Parallel()

	// Load the configuration
	in, err := framework.Load[Config](t)
	require.NoError(t, err)

	// Initialize the blockchain
  e2e-test-inspection-go-test-proposition
	bc, err := blockchain.NewBlockchainNetwork(in.BlockchainA)
	require.NoError(t, err)

	// Initialize Ethereum client
	wsURL := bc.Nodes[0].HostWSUrl
	client, err := ethclient.DialContext(context.Background(), wsURL)
	require.NoError(t, err)

	// Define the deployer's private key
	privateKeyHex := in.Settings.PrivateKey
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" from the key
	require.NoError(t, err)

	// Define signer addresses
	signerAddresses := []common.Address{
		common.HexToAddress("0x1234567890123456789012345678901234567890"),
		common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(31337))
	require.NoError(t, err)

	contractAddress := deployContract(t, client, auth, signerAddresses)

	// Create a shared context
	ctx := &TestContext{
		Client:          client,
		ContractAddress: contractAddress,
		DeployerKey:     crypto.PubkeyToAddress(privateKey.PublicKey),
		SignerAddresses: signerAddresses,
	}

	// Run tests
	t.Run("TestGetConfig", func(t *testing.T) {
		t.Parallel()
		testGetConfig(t, ctx)
	})

	t.Run("TestGetOpCount", func(t *testing.T) {
		t.Parallel()
		testGetOpCount(t, ctx)
	})

	t.Run("TestGetRoot", func(t *testing.T) {
		t.Parallel()
		testGetRoot(t, ctx)
	})

	t.Run("TestGetRootMetadata", func(t *testing.T) {
		t.Parallel()
		testGetRootMetadata(t, ctx)
	})
}

// Helper to deploy the contract
func deployContract(t *testing.T, client *ethclient.Client, auth *bind.TransactOpts, signerAddresses []common.Address) string {
	t.Helper()

	// Deploy the contract
	address, tx, instance, err := bindings.DeployManyChainMultiSig(auth, client)
	require.NoError(t, err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(context.Background(), client, tx)
	require.NoError(t, err, "Failed to mine deployment transaction")

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(auth, signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	require.NoError(t, err, "Failed to set contract configuration")
	_, err = bind.WaitMined(context.Background(), client, tx)
	require.NoError(t, err, "Failed to mine configuration transaction")

	return address.Hex()
}

// TestMCMSConfig checks contract configuration
func testGetConfig(t *testing.T, ctx *TestContext) {
	inspector := evm.NewInspector(ctx.Client)
	config, err := inspector.GetConfig(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get contract configuration")
	require.NotNil(t, config, "Contract configuration is nil")

	// Check first group
	require.Equal(t, uint8(1), config.Quorum, "Quorum does not match")
	require.Equal(t, ctx.SignerAddresses[0], config.Signers[0], "Signers do not match")

	// Check second group
	require.Equal(t, uint8(1), config.GroupSigners[0].Quorum, "Group quorum does not match")
	require.Equal(t, ctx.SignerAddresses[1], config.GroupSigners[0].Signers[0], "Group signers do not match")
}

// TestGetOpCount checks contract operation count
func testGetOpCount(t *testing.T, ctx *TestContext) {
	inspector := evm.NewInspector(ctx.Client)
	opCount, err := inspector.GetOpCount(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get op count")
	require.Equal(t, uint64(0), opCount, "Operation count does not match")
}

// TestGetRoot checks contract operation count
func testGetRoot(t *testing.T, ctx *TestContext) {
	inspector := evm.NewInspector(ctx.Client)
	root, validUntil, err := inspector.GetRoot(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get root from contract")
	require.Equal(t, common.Hash{}, root, "Roots do not match")
	require.Equal(t, uint32(0), validUntil, "ValidUntil does not match")
}

// TestGetRootMetadata checks contract operation count
func testGetRootMetadata(t *testing.T, ctx *TestContext) {
	inspector := evm.NewInspector(ctx.Client)
	metadata, err := inspector.GetRootMetadata(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get root metadata from contract")
	require.Equal(t, metadata.MCMAddress, ctx.ContractAddress, "MCMAddress does not match")
	require.Equal(t, uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
