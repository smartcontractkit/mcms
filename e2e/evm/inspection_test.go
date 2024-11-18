package evm

import (
	"context"
	"crypto/ecdsa"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

// Config struct for blockchain configuration
type Config struct {
	BlockchainA *blockchain.Input `toml:"blockchain_a" validate:"required"`
}

// InspectionTestSuite defines a suite for EVM tests
type InspectionTestSuite struct {
	suite.Suite
	Config       Config
	Blockchain   *blockchain.Output
	Client       *ethclient.Client
	DeployedAddr string
	DeployedIns  *evm.Inspector
	PrivateKey   *ecdsa.PrivateKey
	Auth         *bind.TransactOpts
}

// SetupSuite initializes resources for the suite
func (s *InspectionTestSuite) SetupSuite() {
	// Load the configuration
	in, err := framework.Load[Config](s.T())
	require.NoError(s.T(), err)
	s.Config = *in

	// Initialize the blockchain
	bc, err := blockchain.NewBlockchainNetwork(s.Config.BlockchainA)
	require.NoError(s.T(), err)
	s.Blockchain = bc

	// Initialize Ethereum client
	wsUrl := s.Blockchain.Nodes[0].HostWSUrl
	client, err := ethclient.DialContext(context.Background(), wsUrl)
	require.NoError(s.T(), err, "Failed to connect to Ethereum client")
	s.Client = client

	// Use a pre-funded Anvil account
	privateKeyHex := "0xYOUR_ANVIL_ACCOUNT_PRIVATE_KEY" // Replace with one of Anvil's pre-funded private keys
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:])
	require.NoError(s.T(), err, "Failed to parse private key")
	s.PrivateKey = privateKey

	// Set up transaction options
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337)) // Replace with Anvil's chain ID
	require.NoError(s.T(), err, "Failed to create transaction options")
	s.Auth = auth

	// Deploy contract
	s.deployTestingMCMSContract()
}

// deployTestingMCMSContract handles deploying the ManyChainMultiSig contract using the bindings and storing its address and inspector
func (s *InspectionTestSuite) deployTestingMCMSContract() {
	// Use the `DeployManyChainMultiSig` function from the generated bindings
	address, tx, _, err := bindings.DeployManyChainMultiSig(s.Auth, s.Client)
	require.NoError(s.T(), err, "Failed to deploy ManyChainMultiSig contract")

	// Log the deployed contract address
	log.Printf("Deployed ManyChainMultiSig contract to address: %s", address.Hex())

	// Store the deployed contract address
	s.DeployedAddr = address.Hex()

	// Wait for the transaction to be mined
	bind.WaitMined(context.Background(), s.Client, tx)
	log.Printf("Deployment transaction mined. Contract Address: %s", address.Hex())
}

// TearDownSuite cleans up resources for the suite
func (s *InspectionTestSuite) TearDownSuite() {
	// Add teardown logic if needed
}

// TestMCMSConfig checks contract configuration
func (s *InspectionTestSuite) TestMCMSConfig() {
	config, err := s.DeployedIns.GetConfig(s.DeployedAddr)
	require.NoError(s.T(), err, "Failed to get config")
	require.NotEmpty(s.T(), config, "Config should not be empty")
}

// Entry point for the test suite
func TestEVMTestSuite(t *testing.T) {
	suite.Run(t, new(InspectionTestSuite))
}
