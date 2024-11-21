//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

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

// InspectionTestSuite defines the test suite
type InspectionTestSuite struct {
	suite.Suite
	client          *ethclient.Client
	contractAddress string
	deployerKey     common.Address
	signerAddresses []common.Address
	auth            *bind.TransactOpts
	TestSetup       *TestSetup
}

// SetupSuite runs before the test suite
func (s *InspectionTestSuite) SetupSuite() {
	s.TestSetup = InitializeTestSetup(s.T())
	s.Require().NoError(err, "Failed to initialize test setup")
	s.TestSetup = setup

	in, err := framework.Load[Config](s.T())
	// Get deployer's private key
	privateKeyHex := in.Settings.PrivateKey
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Invalid private key")

	// Define signer addresses
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x1234567890123456789012345678901234567890"),
		common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdef"),
	}

	// Parse ChainID from string to int64
	chainID, ok := new(big.Int).SetString(in.BlockchainA.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth

	s.contractAddress = s.deployContract()
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)
}

// deployContract is a helper to deploy the contract
func (s *InspectionTestSuite) deployContract() string {
	address, tx, instance, err := bindings.DeployManyChainMultiSig(s.auth, s.client)
	require.NoError(s.T(), err, "Failed to deploy contract")

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(context.Background(), s.client, tx)
	require.NoError(s.T(), err, "Failed to mine deployment transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Set configurations
	signerGroups := []uint8{0, 1}   // Two groups: Group 0 and Group 1
	groupQuorums := [32]uint8{1, 1} // Quorum 1 for both groups
	groupParents := [32]uint8{0, 0} // Group 0 is its own parent; Group 1's parent is Group 0
	clearRoot := true

	tx, err = instance.SetConfig(s.auth, s.signerAddresses, signerGroups, groupQuorums, groupParents, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	receipt, err = bind.WaitMined(context.Background(), s.client, tx)
	s.Require().NoError(err, "Failed to mine configuration transaction")
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return address.Hex()
}

// TestGetConfig checks contract configuration
func (s *InspectionTestSuite) TestGetConfig() {
	inspector := evm.NewInspector(s.client)
	config, err := inspector.GetConfig(s.contractAddress)

	s.Require().NoError(err, "Failed to get contract configuration")
	s.Require().NotNil(config, "Contract configuration is nil")

	// Check first group
	s.Require().Equal(uint8(1), config.Quorum, "Quorum does not match")
	s.Require().Equal(s.signerAddresses[0], config.Signers[0], "Signers do not match")

	// Check second group
	s.Require().Equal(uint8(1), config.GroupSigners[0].Quorum, "Group quorum does not match")
	s.Require().Equal(s.signerAddresses[1], config.GroupSigners[0].Signers[0], "Group signers do not match")
}

// TestGetOpCount checks contract operation count
func (s *InspectionTestSuite) TestGetOpCount() {
	inspector := evm.NewInspector(s.client)
	opCount, err := inspector.GetOpCount(s.contractAddress)

	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count does not match")
}

// TestGetRoot checks contract root
func (s *InspectionTestSuite) TestGetRoot() {
	inspector := evm.NewInspector(s.client)
	root, validUntil, err := inspector.GetRoot(s.contractAddress)

	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(common.Hash{}, root, "Roots do not match")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil does not match")
}

// TestGetRootMetadata checks contract root metadata
func (s *InspectionTestSuite) TestGetRootMetadata() {
	inspector := evm.NewInspector(s.client)
	metadata, err := inspector.GetRootMetadata(s.contractAddress)

	s.Require().NoError(err, "Failed to get root metadata from contract")
	s.Require().Equal(metadata.MCMAddress, s.contractAddress, "MCMAddress does not match")
	s.Require().Equal(uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
