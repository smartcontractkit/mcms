//go:build e2e
// +build e2e

package evme2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

// TimelockInspectionTestSuite is a suite of tests for the RBACTimelock contract inspection.
type TimelockInspectionTestSuite struct {
	suite.Suite
	deployerKey      common.Address
	signerAddresses  []common.Address
	auth             *bind.TransactOpts
	publicKey        common.Address
	timelockContract *bindings.RBACTimelock
	e2e.TestSetup
}

func (s *TimelockInspectionTestSuite) granRole(role [32]byte, address common.Address) {
	ctx := context.Background()
	tx, err := s.timelockContract.GrantRole(s.auth, role, address)
	s.Require().NoError(err)
	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
	receipt, err = testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// SetupSuite runs before the test suite
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())
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
	chainID, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	s.Require().NoError(err, "Failed to create transactor")
	s.auth = auth
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	s.Require().True(ok, "Failed to cast public key to ECDSA")

	// Derive the Ethereum address from the public key
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	s.publicKey = address

	s.timelockContract = testutils.DeployTimelockContract(&s.Suite, s.ClientA, s.auth, address.String())
	s.deployerKey = crypto.PubkeyToAddress(privateKey.PublicKey)

	// Grant Some Roles for testing
	// Proposers
	role, err := s.timelockContract.PROPOSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	s.granRole(role, s.signerAddresses[0])
	// Executors
	role, err = s.timelockContract.EXECUTORROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	s.granRole(role, s.signerAddresses[0])
	s.granRole(role, s.signerAddresses[1])

	// By passers
	role, err = s.timelockContract.BYPASSERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	s.granRole(role, s.signerAddresses[1])

	// Cancellers
	role, err = s.timelockContract.CANCELLERROLE(&bind.CallOpts{})
	s.Require().NoError(err)
	s.granRole(role, s.signerAddresses[0])
	s.granRole(role, s.signerAddresses[1])
}

// TestGetProposers gets the list of proposers
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	proposers, err := inspector.GetProposers(ctx, s.timelockContract.Address().Hex())
	s.Require().NoError(err)
	s.Require().Len(proposers, 1)
	s.Require().Equal(s.signerAddresses[0].Hex(), proposers[0])
}

// TestGetExecutors gets the list of executors
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	executors, err := inspector.GetExecutors(ctx, s.timelockContract.Address().Hex())
	s.Require().NoError(err)
	s.Require().Len(executors, 2)
	s.Require().Equal(s.signerAddresses[0].Hex(), executors[0])
	s.Require().Equal(s.signerAddresses[1].Hex(), executors[1])
}

// TestGetBypassers gets the list of bypassers
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	bypassers, err := inspector.GetBypassers(ctx, s.timelockContract.Address().Hex())
	s.Require().NoError(err)
	s.Require().Len(bypassers, 1) // Ensure lengths match
	// Check that all elements of signerAddresses are in proposers
	s.Require().Contains(bypassers, s.signerAddresses[1].Hex())
}

// TestGetCancellers gets the list of cancellers
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	cancellers, err := inspector.GetCancellers(ctx, s.timelockContract.Address().Hex())
	s.Require().NoError(err)
	s.Require().Len(cancellers, 2)
	s.Require().Equal(s.signerAddresses[0].Hex(), cancellers[0])
	s.Require().Equal(s.signerAddresses[1].Hex(), cancellers[1])
}

// TestIsOperation tests the IsOperation method
func (s *TimelockInspectionTestSuite) TestIsOperation() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	// Schedule a test operation
	calls := []bindings.RBACTimelockCall{
		{
			Target: s.signerAddresses[0],
			Value:  big.NewInt(1),
		},
	}
	delay := big.NewInt(3600)
	pred := [32]byte{0x0}
	salt := [32]byte{0x01}
	tx, err := s.timelockContract.ScheduleBatch(s.auth, calls, pred, salt, delay)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash())
	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	opID, err := evm.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperation(ctx, s.timelockContract.Address().Hex(), opID)
	s.Require().NoError(err)
	s.Require().True(isOP)
}

// TestIsOperationPending tests the IsOperationPending method
func (s *TimelockInspectionTestSuite) TestIsOperationPending() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	// Schedule a test operation
	calls := []bindings.RBACTimelockCall{
		{
			Target: s.signerAddresses[0],
			Value:  big.NewInt(2),
		},
	}
	delay := big.NewInt(3600)
	pred, err := evm.HashOperationBatch(calls, [32]byte{0x0}, [32]byte{0x01})
	s.Require().NoError(err)
	salt := [32]byte{0x01}
	tx, err := s.timelockContract.ScheduleBatch(s.auth, calls, pred, salt, delay)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash())
	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	opID, err := evm.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperationPending(ctx, s.timelockContract.Address().Hex(), opID)
	s.Require().NoError(err)
	s.Require().True(isOP)
}

// TestIsOperationReady tests the IsOperationReady and IsOperationDone methods
func (s *TimelockInspectionTestSuite) TestIsOperationReady() {
	ctx := context.Background()
	inspector := evm.NewTimelockInspector(s.ClientA)

	// Schedule a test operation
	calls := []bindings.RBACTimelockCall{
		{
			Target: s.signerAddresses[0],
			Value:  big.NewInt(1),
		},
	}
	delay := big.NewInt(0)
	pred2, err := evm.HashOperationBatch(calls, [32]byte{0x0}, [32]byte{0x01})
	s.Require().NoError(err)
	pred, err := evm.HashOperationBatch(calls, pred2, [32]byte{0x01})
	s.Require().NoError(err)
	salt := [32]byte{0x01}
	tx, err := s.timelockContract.ScheduleBatch(s.auth, calls, pred, salt, delay)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash())
	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	opID, err := evm.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperationReady(ctx, s.timelockContract.Address().Hex(), opID)
	s.Require().NoError(err)
	s.Require().True(isOP)
}

func (s *TimelockInspectionTestSuite) TestIsOperationDone() {
	ctx := context.Background()

	// Deploy a new timelock for this test
	timelockContract := testutils.DeployTimelockContract(&s.Suite, s.ClientA, s.auth, s.publicKey.String())

	// Get the suggested gas price
	gasPrice, err := s.ClientA.SuggestGasPrice(ctx)
	s.Require().NoError(err)
	gasLimit := uint64(30000)
	to := timelockContract.Address()

	pendingNonce, err := s.ClientA.PendingNonceAt(ctx, s.publicKey)
	s.Require().NoError(err)

	txData := &types.LegacyTx{
		Nonce:    pendingNonce,
		To:       &to,
		Value:    big.NewInt(4e15), // 0.004 ETH
		GasPrice: gasPrice.Mul(gasPrice, big.NewInt(10)),
		Gas:      gasLimit,
	}
	tx := types.NewTx(txData)
	// Sign the transaction
	chainID, err := s.ClientA.NetworkID(ctx)
	s.Require().NoError(err)
	privateKeyHex := s.Settings.PrivateKeys[0]
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Strip "0x" prefix
	s.Require().NoError(err)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	s.Require().NoError(err)
	err = s.ClientA.SendTransaction(ctx, signedTx)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash())
	receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, signedTx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Schedule a test operation
	calls := []bindings.RBACTimelockCall{
		{
			Target: s.signerAddresses[1],
			Value:  big.NewInt(1), // 0.001 ETH
			Data:   nil,           // No data, just an ETH transfer
		},
	}
	delay := big.NewInt(0)
	pred := [32]byte{0x0}
	salt := [32]byte{0x01}
	tx, err = timelockContract.ScheduleBatch(s.auth, calls, pred, salt, delay)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash())
	receipt, err = testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
	s.Require().NoError(err)
	s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Use `Eventually` to wait for the transaction to be mined and the operation to be done
	s.Require().Eventually(func() bool {
		// Attempt to execute the batch
		tx, err := timelockContract.ExecuteBatch(s.auth, calls, pred, salt)
		s.Require().NoError(err, "Failed to execute batch")
		s.Require().NotEmpty(tx.Hash(), "Transaction hash is empty")

		// Wait for the transaction to be mined
		receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, tx.Hash())
		s.Require().NoError(err, "Failed to wait for transaction to be mined")
		s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "Transaction was not successful")

		// Check if the operation is done
		inspector := evm.NewTimelockInspector(s.ClientA)
		opID, err := evm.HashOperationBatch(calls, pred, salt)
		s.Require().NoError(err, "Failed to compute operation ID")

		isOpDone, err := inspector.IsOperationDone(ctx, timelockContract.Address().Hex(), opID)
		s.Require().NoError(err, "Failed to check if operation is done")

		return isOpDone
	}, 5*time.Second, 500*time.Millisecond, "Operation was not completed in time")
}
