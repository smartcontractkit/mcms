//go:build e2e

package tone2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	toncommon "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/hash"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// TimelockInspectionTestSuite is a suite of tests for the RBACTimelock contract inspection.
type TimelockInspectionTestSuite struct {
	suite.Suite
	e2e.TestSetup

	wallet       *wallet.Wallet
	timelockAddr *address.Address

	accounts []*address.Address
}

func (s *TimelockInspectionTestSuite) grantRole(role [32]byte, acc *address.Address) {
	ctx := s.T().Context()
	body, err := tlb.ToCell(rbac.GrantRole{
		QueryID: must(tvm.RandomQueryID()),

		Role:    tlbe.NewUint256(new(big.Int).SetBytes(role[:])),
		Account: acc,
	})
	s.Require().NoError(err)

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     s.timelockAddr,
			Amount:      tlb.MustFromTON("0.12"),
			Body:        body,
		},
	}

	tx, _, err := s.wallet.SendWaitTransaction(ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(tx)

	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
}

func (s *TimelockInspectionTestSuite) scheduleBatch(timelockAddr *address.Address, calls []timelock.Call, predecessor, salt common.Hash, delay uint32) {
	ctx := s.T().Context()
	body, err := tlb.ToCell(timelock.ScheduleBatch{
		QueryID: must(tvm.RandomQueryID()),

		Calls:       calls,
		Predecessor: tlbe.NewUint256(predecessor.Big()),
		Salt:        tlbe.NewUint256(salt.Big()),
		Delay:       delay,
	})
	s.Require().NoError(err)

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     timelockAddr,
			Amount:      tlb.MustFromTON("0.3"),
			Body:        body,
		},
	}

	tx, _, err := s.wallet.SendWaitTransaction(ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(tx)

	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
}

func (s *TimelockInspectionTestSuite) deployTimelockContract(id uint32) (*address.Address, error) {
	ctx := s.T().Context()
	amount := tlb.MustFromTON("1.5") // TODO (ton): high gas

	data := timelock.EmptyDataFrom(id)
	// When deploying the contract, send the Init message to initialize the Timelock contract
	// Admin will get all roles (not required, just for testing)
	addrs := []toncommon.AddressWrap{
		{Val: s.wallet.Address()},
	}

	body := timelock.Init{
		QueryID:                  0,
		MinDelay:                 0,
		Admin:                    s.wallet.Address(),
		Proposers:                addrs,
		Executors:                addrs,
		Cancellers:               addrs,
		Bypassers:                addrs,
		ExecutorRoleCheckEnabled: true,
		OpFinalizationTimeout:    0,
	}

	return DeployTimelockContract(ctx, s.TonClient, s.wallet, amount, data, body)
}

// SetupSuite runs before the test suite
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	// Generate few test wallets
	chainID := cselectors.TON_LOCALNET.ChainID
	client := s.TonClient
	s.accounts = []*address.Address{
		must(tvm.NewRandomV5R1TestWallet(client, chainID)).Address(),
		must(tvm.NewRandomV5R1TestWallet(client, chainID)).Address(),
	}

	// Sort accounts to have deterministic order
	slices.SortFunc(s.accounts, func(a, b *address.Address) int {
		return bytes.Compare(a.Data(), b.Data())
	})

	var err error
	s.wallet, err = tvm.MyLocalTONWalletDefault(client)
	s.Require().NoError(err)

	// Deploy Timelock contract
	s.timelockAddr, err = s.deployTimelockContract(hash.CRC32("test.timelock_inspection.timelock"))
	s.Require().NoError(err)

	// Grant Some Roles for testing
	// Proposers
	role := [32]byte(timelock.RoleProposer.Bytes())
	s.Require().NoError(err)
	s.grantRole(role, s.accounts[0])
	// Executors
	role = [32]byte(timelock.RoleExecutor.Bytes())
	s.Require().NoError(err)
	s.grantRole(role, s.accounts[0])
	s.grantRole(role, s.accounts[1])

	// Bypassers
	role = [32]byte(timelock.RoleBypasser.Bytes())
	s.Require().NoError(err)
	s.grantRole(role, s.accounts[1])

	// Cancellers
	role = [32]byte(timelock.RoleCanceller.Bytes())
	s.Require().NoError(err)
	s.grantRole(role, s.accounts[0])
	s.grantRole(role, s.accounts[1])
}

// TestGetProposers gets the list of proposers
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	proposers, err := inspector.GetProposers(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(proposers, 2)
	s.Require().Equal(s.accounts[0].String(), proposers[0])
}

// TestGetExecutors gets the list of executors
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	executors, err := inspector.GetExecutors(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(executors, 3)
	s.Require().Equal(s.accounts[0].String(), executors[0])
	s.Require().Equal(s.accounts[1].String(), executors[1])
}

// TestGetBypassers gets the list of bypassers
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	bypassers, err := inspector.GetBypassers(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(bypassers, 2) // Ensure lengths match
	// Check that all elements of signerAddresses are in proposers
	s.Require().Contains(bypassers, s.accounts[1].String())
}

// TestGetCancellers gets the list of cancellers
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	cancellers, err := inspector.GetCancellers(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(cancellers, 3)
	s.Require().Equal(s.accounts[0].String(), cancellers[0])
	s.Require().Equal(s.accounts[1].String(), cancellers[1])
}

// TestIsOperation tests the IsOperation method
func (s *TimelockInspectionTestSuite) TestIsOperation() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	// Schedule a test operation
	calls := []timelock.Call{
		{
			Target: s.accounts[0],
			Value:  tlb.MustFromTON("0.1"), // TON implementation enforces min value per call
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 3600
	pred := common.Hash([32]byte{0x0})
	salt := common.Hash([32]byte{0x01})
	s.scheduleBatch(s.timelockAddr, calls, pred, salt, uint32(delay))

	opID, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperation(ctx, s.timelockAddr.String(), opID)
	s.Require().NoError(err)
	s.Require().True(isOP)
}

// TestIsOperationPending tests the IsOperationPending method
func (s *TimelockInspectionTestSuite) TestIsOperationPending() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	// Schedule a test operation
	calls := []timelock.Call{
		{
			Target: s.accounts[0],
			Value:  tlb.MustFromTON("0.1"), // TON implementation enforces min value per call
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 3600
	salt := common.Hash([32]byte{0x01})
	pred, err := mcmston.HashOperationBatch(calls, [32]byte{0x0}, salt)
	s.Require().NoError(err)
	s.scheduleBatch(s.timelockAddr, calls, pred, salt, uint32(delay))

	opID, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperationPending(ctx, s.timelockAddr.String(), opID)
	s.Require().NoError(err)
	s.Require().True(isOP)
}

// TestIsOperationReady tests the IsOperationReady and IsOperationDone methods
func (s *TimelockInspectionTestSuite) TestIsOperationReady() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	// Schedule a test operation
	calls := []timelock.Call{
		{
			Target: s.accounts[0],
			Value:  tlb.MustFromTON("0.1"), // TON implementation enforces min value per call
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 0
	salt := common.Hash([32]byte{0x01})
	pred2, err := mcmston.HashOperationBatch(calls, [32]byte{0x0}, salt)
	s.Require().NoError(err)
	pred, err := mcmston.HashOperationBatch(calls, pred2, salt)
	s.Require().NoError(err)
	s.scheduleBatch(s.timelockAddr, calls, pred, salt, uint32(delay))

	opID, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperationReady(ctx, s.timelockAddr.String(), opID)
	s.Require().NoError(err)
	s.Require().True(isOP)
}

func (s *TimelockInspectionTestSuite) TestIsOperationDone() {
	ctx := s.T().Context()

	// Deploy a new timelock for this test
	newTimelockAddr, err := s.deployTimelockContract(hash.CRC32("test.timelock_inspection.timelock.1"))
	s.Require().NoError(err)

	// Schedule a test operation
	calls := []timelock.Call{
		{
			Target: s.accounts[1],
			Value:  tlb.MustFromTON("0.1"), // TON implementation enforces min value per call
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 0
	pred := common.Hash([32]byte{0x0})
	salt := common.Hash([32]byte{0x01})

	id, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)

	s.scheduleBatch(newTimelockAddr, calls, pred, salt, uint32(delay))

	inspector := mcmston.NewTimelockInspector(s.TonClient)
	isOp, err := inspector.IsOperation(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation exists")
	s.Require().True(isOp, "Operation should exist")

	isOpPending, err := inspector.IsOperationPending(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation is pending")
	s.Require().True(isOpPending, "Operation should be pending")

	isOpReady, err := inspector.IsOperationReady(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation is ready")
	s.Require().True(isOpReady, "Operation should be ready")

	isOpDone, err := inspector.IsOperationDone(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation is done")
	s.Require().False(isOpDone, "Operation should not be done yet")

	// Attempt to execute the batch
	executor, err := mcmston.NewTimelockExecutor(s.TonClient, s.wallet, tlb.MustFromTON("0.2"))
	s.Require().NoError(err, "Failed to create TimelockExecutor")

	bop := types.BatchOperation{
		ChainSelector: types.ChainSelector(cselectors.TON_LOCALNET.Selector),
		Transactions: []types.Transaction{
			{
				To:               s.accounts[1].String(),
				Data:             cell.BeginCell().EndCell().ToBOC(),
				AdditionalFields: json.RawMessage(fmt.Sprintf(`{"value": %d}`, tlb.MustFromTON("0.1").Nano().Uint64())),
			},
		},
	}

	// Test same ID
	_calls, err := mcmston.ConvertBatchToCalls(bop)
	s.Require().NoError(err, "Failed to convert batch to calls")

	_id, err := mcmston.HashOperationBatch(_calls, pred, salt)
	s.Require().NoError(err, "Failed to compute operation ID")
	s.Require().Equal(id, _id, "Operation IDs do not match")

	res, err := executor.Execute(ctx, bop, newTimelockAddr.String(), pred, salt)
	s.Require().NoError(err, "Failed to execute batch")
	s.Require().NotEmpty(res.Hash, "Transaction hash is empty")

	// Wait for the transaction to be mined
	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx.Description)

	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

	// Check the operation (still) exists
	isOp, err = inspector.IsOperation(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation exists")
	s.Require().True(isOp, "Operation should exist")

	// Check the operation is NOT pending anymore
	isOpPending, err = inspector.IsOperationPending(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation is pending")
	s.Require().False(isOpPending, "Operation should NOT be pending")

	// Check the operation is NOT done
	isOpDone, err = inspector.IsOperationDone(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation is done")
	s.Require().False(isOpDone, "Operation should NOT be done (in error state)")

	// Check the operation is in error state (bounced from an uninitialized account)
	tonInspector, ok := inspector.(*mcmston.TimelockInspector)
	s.Require().True(ok, "Inspector is not of type TimelockInspector")

	isOpError, err := tonInspector.IsOperationError(ctx, newTimelockAddr.String(), id)
	s.Require().NoError(err, "Failed to check if operation is in error state")
	s.Require().True(isOpError, "Operation should be in error state")
}

// TestGetMinDelay tests the GetMinDelay method
func (s *TimelockInspectionTestSuite) TestGetMinDelay() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	delay, err := inspector.GetMinDelay(ctx, s.timelockAddr.String())
	s.Require().NoError(err, "Failed to get min delay")
	s.Require().EqualValues(0, delay)
}
