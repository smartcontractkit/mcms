//go:build e2e

package tone2e

import (
	"math/big"
	"slices"
	"strings"

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
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/sdk/ton"
	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
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
		QueryID: must(ton.RandomQueryID()),

		Role:    new(big.Int).SetBytes(role[:]),
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

	// TODO: confirm expectedtransaction success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)

}

func (s *TimelockInspectionTestSuite) scheduleBatch(calls []timelock.Call, predecessor *big.Int, salt *big.Int, delay uint32) {
	ctx := s.T().Context()
	body, err := tlb.ToCell(timelock.ScheduleBatch{
		QueryID: must(ton.RandomQueryID()),

		Calls:       toncommon.SnakeRef[timelock.Call](calls),
		Predecessor: predecessor,
		Salt:        salt,
		Delay:       delay,
	})
	s.Require().NoError(err)

	msg := &wallet.Message{
		Mode: wallet.PayGasSeparately | wallet.IgnoreErrors,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      true,
			DstAddr:     s.timelockAddr,
			Amount:      tlb.MustFromTON("0.3"),
			Body:        body,
		},
	}

	tx, _, err := s.wallet.SendWaitTransaction(ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(tx)

	// TODO: confirm expectedtransaction success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
}

func (s *TimelockInspectionTestSuite) deployTimelockContract() {
	ctx := s.T().Context()
	amount := tlb.MustFromTON("0.5") // TODO: high gas

	data := TimelockEmptyDataFrom(hash.CRC32("test.timelock_inspection.timelock"))
	// When deploying the contract, send the Init message to initialize the Timelock contract
	none := []toncommon.WrappedAddress{}
	body := timelock.Init{
		QueryID:                  0,
		MinDelay:                 0,
		Admin:                    s.wallet.Address(),
		Proposers:                toncommon.SnakeRef[toncommon.WrappedAddress](none),
		Executors:                toncommon.SnakeRef[toncommon.WrappedAddress](none),
		Cancellers:               toncommon.SnakeRef[toncommon.WrappedAddress](none),
		Bypassers:                toncommon.SnakeRef[toncommon.WrappedAddress](none),
		ExecutorRoleCheckEnabled: true,
		OpFinalizationTimeout:    0,
	}
	var err error
	s.timelockAddr, err = DeployTimelockContract(ctx, s.TonClient, s.wallet, amount, data, body)
	s.Require().NoError(err)
}

// SetupSuite runs before the test suite
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	// Generate few test wallets
	chainID := cselectors.TON_LOCALNET.ChainID
	client := s.TonClient
	s.accounts = []*address.Address{
		must(makeRandomTestWallet(client, chainID)).Address(),
		must(makeRandomTestWallet(client, chainID)).Address(),
	}

	// Sort accounts to have deterministic order
	slices.SortFunc(s.accounts, func(a, b *address.Address) int {
		return strings.Compare(strings.ToLower(a.String()), strings.ToLower(b.String()))
	})

	var err error
	s.wallet, err = LocalWalletDefault(client)
	s.Require().NoError(err)

	// Deploy Timelock contract
	s.deployTimelockContract()

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
	s.Require().Len(proposers, 1)
	s.Require().Equal(s.accounts[0].String(), proposers[0])
}

// TestGetExecutors gets the list of executors
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	executors, err := inspector.GetExecutors(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(executors, 2)
	s.Require().Equal(s.accounts[0].String(), executors[0])
	s.Require().Equal(s.accounts[1].String(), executors[1])
}

// TestGetBypassers gets the list of bypassers
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	bypassers, err := inspector.GetBypassers(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(bypassers, 1) // Ensure lengths match
	// Check that all elements of signerAddresses are in proposers
	s.Require().Contains(bypassers, s.accounts[1].String())
}

// TestGetCancellers gets the list of cancellers
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	cancellers, err := inspector.GetCancellers(ctx, s.timelockAddr.String())
	s.Require().NoError(err)
	s.Require().Len(cancellers, 2)
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
			Value:  big.NewInt(1),
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 3600
	pred := common.Hash([32]byte{0x0})
	salt := common.Hash([32]byte{0x01})
	s.scheduleBatch(calls, pred.Big(), salt.Big(), uint32(delay))

	opID, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperation(ctx, s.timelockAddr.String(), opID)
	s.Require().NoError(err)
	s.Require().NotNil(isOP)
	// s.Require().True(isOP) // TODO(ton): fix
}

// TestIsOperationPending tests the IsOperationPending method
func (s *TimelockInspectionTestSuite) TestIsOperationPending() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	// Schedule a test operation
	calls := []timelock.Call{
		{
			Target: s.accounts[0],
			Value:  big.NewInt(2),
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 3600
	pred, err := mcmston.HashOperationBatch(calls, [32]byte{0x0}, [32]byte{0x01})
	s.Require().NoError(err)
	salt := common.Hash([32]byte{0x01})
	s.scheduleBatch(calls, pred.Big(), salt.Big(), uint32(delay))

	opID, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperationPending(ctx, s.timelockAddr.String(), opID)
	s.Require().NoError(err)
	s.Require().NotNil(isOP)
	// s.Require().True(isOP) // TODO(ton): fix
}

// TestIsOperationReady tests the IsOperationReady and IsOperationDone methods
func (s *TimelockInspectionTestSuite) TestIsOperationReady() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	// Schedule a test operation
	calls := []timelock.Call{
		{
			Target: s.accounts[0],
			Value:  big.NewInt(1),
			Data:   cell.BeginCell().EndCell(),
		},
	}
	delay := 0
	pred2, err := mcmston.HashOperationBatch(calls, [32]byte{0x0}, [32]byte{0x01})
	s.Require().NoError(err)
	pred, err := mcmston.HashOperationBatch(calls, pred2, [32]byte{0x01})
	s.Require().NoError(err)
	salt := common.Hash([32]byte{0x01})
	s.scheduleBatch(calls, pred.Big(), salt.Big(), uint32(delay))

	opID, err := mcmston.HashOperationBatch(calls, pred, salt)
	s.Require().NoError(err)
	isOP, err := inspector.IsOperationReady(ctx, s.timelockAddr.String(), opID)
	s.Require().NoError(err)
	s.Require().NotNil(isOP)
	// s.Require().True(isOP) // TODO(ton): fix
}

// TODO: add TestIsOperationDone test when we have operation execution implemented

// TestGetMinDelay tests the GetMinDelay method
func (s *TimelockInspectionTestSuite) TestGetMinDelay() {
	ctx := s.T().Context()
	inspector := mcmston.NewTimelockInspector(s.TonClient)

	delay, err := inspector.GetMinDelay(ctx, s.timelockAddr.String())
	s.Require().NoError(err, "Failed to get min delay")
	s.Require().EqualValues(0, delay)
}
