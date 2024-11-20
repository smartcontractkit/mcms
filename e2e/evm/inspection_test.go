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

	// Initialize the test context
	ctx := setupTestEnvironment(t)

	// Run tests
	t.Run("TestGetConfig", func(t *testing.T) {
		t.Parallel()
		ctx.TestGetConfig(t)
	})

	t.Run("TestGetOpCount", func(t *testing.T) {
		t.Parallel()
		ctx.TestGetOpCount(t)
	})

	t.Run("TestGetRoot", func(t *testing.T) {
		t.Parallel()
		ctx.TestGetRoot(t)
	})

	t.Run("TestGetRootMetadata", func(t *testing.T) {
		t.Parallel()
		ctx.TestGetRootMetadata(t)
	})
}

func setupTestEnvironment(t *testing.T) *TestContext {
	t.Helper()

	// Load the configuration
	in, err := framework.Load[Config](t)
	require.NoError(t, err, "Failed to load configuration")

	// Initialize the blockchain
	bc, err := blockchain.NewBlockchainNetwork(in.BlockchainA)
	require.NoError(t, err, "Failed to initialize blockchain network")

	// Initialize Ethereum client
	wsURL := bc.Nodes[0].HostWSUrl
	client, err := ethclient.DialContext(context.Background(), wsURL)
	require.NoError(t, err, "Failed to initialize Ethereum client")

	// Get deployer's private key
	privateKeyHex := in.Settings.PrivateKey
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	require.NoError(t, err, "Invalid private key")

	// Define signer addresses
	signerAddresses := []common.Address{
		common.HexToAddress("0x1234567890123456789012345678901234567890"),
		common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(31337))
	require.NoError(t, err, "Failed to create transactor")

	contractAddress := deployContract(t, client, auth, signerAddresses)

	// Return the test context
	return &TestContext{
		Client:          client,
		ContractAddress: contractAddress,
		DeployerKey:     crypto.PubkeyToAddress(privateKey.PublicKey),
		SignerAddresses: signerAddresses,
	}
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
func (ctx *TestContext) TestGetConfig(t *testing.T) {
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
func (ctx *TestContext) TestGetOpCount(t *testing.T) {
	inspector := evm.NewInspector(ctx.Client)
	opCount, err := inspector.GetOpCount(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get op count")
	require.Equal(t, uint64(0), opCount, "Operation count does not match")
}

// TestGetRoot checks contract operation count
func (ctx *TestContext) TestGetRoot(t *testing.T) {
	inspector := evm.NewInspector(ctx.Client)
	root, validUntil, err := inspector.GetRoot(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get root from contract")
	require.Equal(t, common.Hash{}, root, "Roots do not match")
	require.Equal(t, uint32(0), validUntil, "ValidUntil does not match")
}

// TestGetRootMetadata checks contract operation count
func (ctx *TestContext) TestGetRootMetadata(t *testing.T) {
	inspector := evm.NewInspector(ctx.Client)
	metadata, err := inspector.GetRootMetadata(ctx.ContractAddress)

	require.NoError(t, err, "Failed to get root metadata from contract")
	require.Equal(t, metadata.MCMAddress, ctx.ContractAddress, "MCMAddress does not match")
	require.Equal(t, uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
