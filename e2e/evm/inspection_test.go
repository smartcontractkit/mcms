package evm

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

// Config struct for blockchain configuration
type Config struct {
	InputChainConf blockchain.Input `toml:"evm_config" validate:"required"`
	TestData       TestData         `toml:"evm_test_data" validate:"required"`
}

// TestData holds the mnemonic used for testing
type TestData struct {
	Mnemonic string `toml:"mnemonic" validate:"required"`
}

// InspectionTestSuite defines a suite for EVM tests
type InspectionTestSuite struct {
	suite.Suite
	config          Config
	blockchain      *blockchain.Output
	client          *ethclient.Client
	deployedAddr    string
	instance        *bindings.ManyChainMultiSig
	privateKey      *ecdsa.PrivateKey
	auth            *bind.TransactOpts
	signerAddresses []common.Address
}

// SetupSuite initializes resources for the suite
func (s *InspectionTestSuite) SetupSuite() {
	// Load the configuration
	in, err := framework.Load[Config](s.T())
	s.Require().NoError(err)
	s.config = *in

	// Initialize the blockchain
	bc, err := blockchain.NewBlockchainNetwork(&blockchain.Input{
		ChainID: s.config.InputChainConf.ChainID,
		Image:   s.config.InputChainConf.Image,
		Port:    s.config.InputChainConf.Port,
		Type:    s.config.InputChainConf.Type,
	})
	s.Require().NoError(err)
	s.blockchain = bc

	// Initialize Ethereum client
	wsURL := s.blockchain.Nodes[0].HostWSUrl
	client, err := ethclient.DialContext(context.Background(), wsURL)
	s.Require().NoError(err, "Failed to connect to Ethereum client")
	s.client = client

	// Derive the private key from the mnemonic
	privateKey, address, err := derivePrivateKeyFromMnemonic(s.config.TestData.Mnemonic, 0)
	s.Require().NoError(err, "Failed to derive private key")
	s.privateKey = privateKey
	s.deployedAddr = address.Hex()

	// Set up transaction options
	chainID, ok := new(big.Int).SetString(s.config.InputChainConf.ChainID, 10)
	s.Require().True(ok, "Invalid chain ID")
	s.auth, err = bind.NewKeyedTransactorWithChainID(s.privateKey, chainID)
	s.Require().NoError(err, "Failed to create transaction options")

	// Deploy your contract
	s.deployTestingContract()
}

// deployTestingContract handles deploying your contract and storing its address
func (s *InspectionTestSuite) deployTestingContract() {
	// Replace the following with your contract deployment code

	address, tx, instance, err := bindings.DeployManyChainMultiSig(s.auth, s.client)
	s.Require().NoError(err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(context.Background(), s.client, tx)
	s.Require().NoError(err, "Failed to mine deployment transaction")

	s.deployedAddr = address.Hex()
	s.instance = instance

	// Set configurations
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x1234567890123456789012345678901234567890"),
		common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
	}
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = s.instance.SetConfig(s.auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	// Wait for the transaction to be mined
	_, err = bind.WaitMined(context.Background(), s.client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
}

// TearDownSuite cleans up resources for the suite
func (s *InspectionTestSuite) TearDownSuite() {
	// Add teardown logic if needed
	s.client.Close()
}

// TestMCMSConfig checks contract configuration
func (s *InspectionTestSuite) TestMCMSConfig() {
	inspector := evm.NewInspector(s.client)
	config, err := inspector.GetConfig(s.deployedAddr)

	s.Require().NoError(err, "Failed to get contract configuration")
	s.NotNil(config, "Contract configuration is nil")
	// Check first group
	s.Equal(uint8(1), config.Quorum, "Quorum does not match")
	s.Equal([]common.Address{s.signerAddresses[0]}, config.Signers, "Signers do not match")
	// Check second group
	s.Equal(uint8(1), config.GroupSigners[0].Quorum, "Group quorum does not match")
	s.Equal([]common.Address{s.signerAddresses[1]}, config.GroupSigners[0].Signers, "Group signers do not match")
}

// TestGetOpCount checks contract operation count
func (s *InspectionTestSuite) TestGetOpCount() {
	inspector := evm.NewInspector(s.client)
	opCount, err := inspector.GetOpCount(s.deployedAddr)

	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "op count do not match")
}

// TestGetRoot checks contract operation count
func (s *InspectionTestSuite) TestGetRoot() {
	inspector := evm.NewInspector(s.client)
	root, validUntil, err := inspector.GetRoot(s.deployedAddr)

	s.Require().NoError(err, "Failed to get root from contract")
	s.Equal(common.Hash{}, root, "roots do not match")
	s.Equal(uint32(0), validUntil, "validUntil do not match")
}

// TestGetRootMetadata checks contract operation count
func (s *InspectionTestSuite) TestGetRootMetadata() {
	inspector := evm.NewInspector(s.client)
	metadata, err := inspector.GetRootMetadata(s.deployedAddr)

	s.Require().NoError(err, "Failed to get root from contract")
	s.Equal(metadata.MCMAddress, s.deployedAddr, "roots do not match")
	s.Equal(uint64(0), metadata.StartingOpCount, "validUntil do not match")
}

// Entry point for the test suite
func TestEVMTestSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(InspectionTestSuite))
}
