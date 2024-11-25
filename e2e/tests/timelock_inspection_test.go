//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

// TimelockInspectionTestSuite is a suite of tests for the RBACTimelock contract inspection.
type TimelockInspectionTestSuite struct {
	suite.Suite
	mcmsContract     *bindings.ManyChainMultiSig
	deployerKey      common.Address
	signerAddresses  []common.Address
	auth             *bind.TransactOpts
	timelockContract *bindings.RBACTimelock
	TestSetup
}

// SetupSuite runs before the test suite
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.TestSetup = *InitializeSharedTestSetup(s.T())
	// Get deployer's private key
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err, "Invalid private key")

	// Define signer addresses
	s.signerAddresses = []common.Address{
		common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"),
		common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	}

	// Parse ChainID from string to int64
	chainID, ok := new(big.Int).SetString(s.BlockchainA.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	s.Require().True(ok, "Failed to cast public key to ECDSA")

	// Derive the Ethereum address from the public key
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Display the address in hexadecimal format
	fmt.Printf("Ethereum Address: %s\n", address.Hex())

	s.timelockContract = testutils.DeployTimelockContract(&s.Suite, s.Client, s.auth, address.String())
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)

	// Grant Some Roles for testing
	// Proposers
	role, err := s.timelockContract.PROPOSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	tx, err := s.timelockContract.GrantRole(s.auth, role, s.signerAddresses[0])
	s.Require().NoError(err)
	receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.Client, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Executors
	role, err = s.timelockContract.EXECUTORROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	tx, err = s.timelockContract.GrantRole(s.auth, role, s.signerAddresses[0])
	s.Require().NoError(err)
	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	tx, err = s.timelockContract.GrantRole(s.auth, role, s.signerAddresses[1])
	s.Require().NoError(err)
	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// By passers
	role, err = s.timelockContract.BYPASSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	tx, err = s.timelockContract.GrantRole(s.auth, role, s.signerAddresses[1])
	s.Require().NoError(err)
	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Cancellers
	role, err = s.timelockContract.CANCELLERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	tx, err = s.timelockContract.GrantRole(s.auth, role, s.signerAddresses[0])
	s.Require().NoError(err)
	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	tx, err = s.timelockContract.GrantRole(s.auth, role, s.signerAddresses[1])
	s.Require().NoError(err)
	receipt, err = testutils.WaitMinedWithTxHash(context.Background(), s.Client, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// TestGetProposers gets the list of proposers
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	inspector := evm.NewTimelockInspector(s.Client)
	proposers, err := inspector.GetProposers(s.timelockContract.Address().Hex())
	s.Require().NoError(err, "Failed to get proposers")
	s.Require().Equal(1, len(proposers), "Expected 0 proposers")
	s.Require().Equal(s.signerAddresses[0], proposers[0])
}

// TestGetExecutors gets the list of executors
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	inspector := evm.NewTimelockInspector(s.Client)
	proposers, err := inspector.GetExecutors(s.timelockContract.Address().Hex())
	s.Require().NoError(err, "Failed to get proposers")
	s.Require().Equal(2, len(proposers), "Expected 0 proposers")
	s.Require().Equal(s.signerAddresses[0], proposers[0])
	s.Require().Equal(s.signerAddresses[1], proposers[1])
}

// TestGetBypassers gets the list of executors
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	inspector := evm.NewTimelockInspector(s.Client)
	proposers, err := inspector.GetBypassers(s.timelockContract.Address().Hex())
	s.Require().NoError(err, "Failed to get proposers")
	s.Require().Equal(1, len(proposers), "Expected 0 proposers")
	s.Require().Equal(s.signerAddresses[1], proposers[0])

}

// TestGetCancellers gets the list of executors
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	inspector := evm.NewTimelockInspector(s.Client)
	proposers, err := inspector.GetCancellers(s.timelockContract.Address().Hex())
	s.Require().NoError(err, "Failed to get proposers")
	s.Require().Equal(2, len(proposers), "Expected 0 proposers")
	s.Require().Equal(s.signerAddresses[0], proposers[0])
	s.Require().Equal(s.signerAddresses[1], proposers[1])
}
