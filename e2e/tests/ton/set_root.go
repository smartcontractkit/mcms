//go:build e2e
// +build e2e

package tone2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/hash"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/wrappers"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	mcmslib "github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

const (
	ADDR_TIMELOCK = "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8" // static mock address
)

// SetRootTestSuite tests the SetRoot functionality
type SetRootTestSuite struct {
	suite.Suite
	e2e.TestSetup

	wallet   *wallet.Wallet
	mcmsAddr string

	signers       []testutils.ECDSASigner
	chainSelector types.ChainSelector
	accounts      []*address.Address
}

// SetupSuite runs before the test suite
func (s *SetRootTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	// Generate few test signers
	s.signers = testutils.MakeNewECDSASigners(2)

	// Generate few test wallets
	var chainID = chaintest.Chain7TONID
	var client *ton.APIClient = nil
	s.accounts = []*address.Address{
		must(makeRandomTestWallet(client, chainID)).Address(),
		must(makeRandomTestWallet(client, chainID)).Address(),
	}

	var err error
	s.wallet, err = LocalWalletDefault(s.TonClient)
	s.Require().NoError(err)

	s.deployMCMSContract()

	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.TonBlockchain.ChainID, s.TonBlockchain.Family)
	s.Require().NoError(err)
	s.chainSelector = types.ChainSelector(chainDetails.ChainSelector)
}

// TODO: duplicated with SetConfigTestSuite
func (s *SetRootTestSuite) deployMCMSContract() {
	amount := tlb.MustFromTON("0.05")
	msgBody := cell.BeginCell().EndCell() // empty cell, top up

	contractPath := filepath.Join(os.Getenv(EnvPathContracts), PathContractsMCMS)
	contractCode, err := wrappers.ParseCompiledContract(contractPath)
	s.Require().NoError(err)

	chainId, err := strconv.ParseInt(s.TonBlockchain.ChainID, 10, 64)
	s.Require().NoError(err)
	contractData, err := tlb.ToCell(MCMSEmptyDataFrom(hash.CRC32("mcms-test"), s.wallet.Address(), chainId))
	s.Require().NoError(err)

	// TODO: extract .WaitTrace(tx) functionality and use here instead of wrapper
	client := tracetracking.NewSignedAPIClient(s.TonClient, *s.wallet)
	contract, _, err := wrappers.Deploy(s.T().Context(), &client, contractCode, contractData, amount, msgBody)
	s.Require().NoError(err)
	addr := contract.Address

	// workchain := int8(-1)
	// addr, tx, _, err := s.wallet.DeployContractWaitTransaction(s.T().Context(), amount, msgBody, contractCode, contractData, workchain)
	s.Require().NoError(err)
	// s.Require().NotNil(tx)

	s.mcmsAddr = addr.String()

	// Set configuration
	configurerTON, err := mcmston.NewConfigurer(s.wallet, amount)
	s.Require().NoError(err)

	config := &types.Config{
		Quorum: 1,
		Signers: []common.Address{
			s.signers[0].Address(),
			s.signers[1].Address(),
		},
		GroupSigners: []types.Config{
			{
				Quorum: 1,
				Signers: []common.Address{
					s.signers[0].Address(),
					s.signers[1].Address(),
				},
				GroupSigners: []types.Config{},
			},
		},
	}

	clearRoot := true
	tx, err := configurerTON.SetConfig(s.T().Context(), s.mcmsAddr, config, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	s.Require().NotNil(tx)

	// TODO: ton.WaitTrace(tx)
	// receipt, err = bind.WaitMined(context.Background(), s.ClientA, tx)
	// s.Require().NoError(err, "Failed to mine configuration transaction")
	// s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// TestSetRootProposal sets the root of the MCMS contract
func (s *SetRootTestSuite) TestSetRootProposal() {
	ctx := context.Background()
	builder := mcmslib.NewProposalBuilder()
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		AddChainMetadata(
			s.chainSelector,
			types.ChainMetadata{MCMAddress: s.mcmsAddr},
		).
		AddOperation(types.Operation{
			ChainSelector: s.chainSelector,
			Transaction: types.Transaction{
				To:               s.accounts[0].String(),
				Data:             cell.BeginCell().EndCell().ToBOC(),
				AdditionalFields: json.RawMessage(`{"value": 0}`),
			},
		})
	proposal, err := builder.Build()
	s.Require().NoError(err)

	// Sign the proposal
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: mcmston.NewInspector(s.TonClient, mcmston.NewConfigTransformer()),
	}
	signable, err := mcmslib.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[0].Key))
	s.Require().NoError(err)

	// TODO: errors on TON, getter called before setter
	// Validate the signatures
	// quorumMet, err := signable.ValidateSignatures(ctx)
	// s.Require().NoError(err)
	// s.Require().True(quorumMet)

	// Create the chain MCMS proposal executor
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*mcmston.Encoder)

	executor, err := mcmston.NewExecutor(encoder, s.TonClient, s.wallet, tlb.MustFromTON("0.1"))
	s.Require().NoError(err)
	executorsMap := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcmslib.NewExecutable(proposal, executorsMap)
	s.Require().NoError(err)

	// Call SetRoot
	tx, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	// TODO: ton.WaitTrace(tx)
	// receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(tx.Hash))
	// s.Require().NoError(err, "Failed to mine deployment transaction")
	// s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// TestSetRootTimelockProposal sets the root of the MCMS contract from a timelock proposal type.
func (s *SetRootTestSuite) TestSetRootTimelockProposal() {
	ctx := context.Background()

	builder := mcmslib.NewTimelockProposalBuilder()
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		SetAction(types.TimelockActionSchedule).
		SetDelay(types.MustParseDuration("24h")).
		SetTimelockAddresses(map[types.ChainSelector]string{
			s.chainSelector: ADDR_TIMELOCK,
		}).
		AddChainMetadata(
			s.chainSelector,
			types.ChainMetadata{MCMAddress: s.mcmsAddr},
		).
		AddOperation(types.BatchOperation{
			ChainSelector: s.chainSelector,
			Transactions: []types.Transaction{
				{
					To:               s.accounts[0].String(),
					Data:             cell.BeginCell().MustStoreSlice([]byte{0x01}, 8).EndCell().ToBOC(),
					AdditionalFields: json.RawMessage(`{"value": 3}`),
				},
				{
					To:               s.accounts[1].String(),
					Data:             cell.BeginCell().MustStoreSlice([]byte{0x02}, 8).EndCell().ToBOC(),
					AdditionalFields: json.RawMessage(`{"value": 4}`),
				},
			},
		})
	proposalTimelock, err := builder.Build()
	s.Require().NoError(err)

	proposal, _, err := proposalTimelock.Convert(ctx, map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: mcmston.NewTimelockConverter(),
	})
	s.Require().NoError(err)

	// Sign proposal
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: mcmston.NewInspector(s.TonClient, mcmston.NewConfigTransformer()),
	}
	signable, err := mcmslib.NewSignable(&proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[1].Key))
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*mcmston.Encoder)

	executor, err := mcmston.NewExecutor(encoder, s.TonClient, s.wallet, tlb.MustFromTON("0.1"))
	s.Require().NoError(err)
	executorsMap := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// TODO: no simulation on TON
	// // Prepare and execute simulation
	// simulator, err := evm.NewSimulator(encoder, s.ClientA)
	// s.Require().NoError(err, "Failed to create simulator")
	// simulators := map[types.ChainSelector]sdk.Simulator{
	// 	s.chainSelector: simulator,
	// }
	// signable.SetSimulators(simulators)
	// err = signable.Simulate(ctx)
	// s.Require().NoError(err)

	// Create the chain MCMS proposal executor
	executable, err := mcmslib.NewExecutable(&proposal, executorsMap)
	s.Require().NoError(err)
	// Call SetRoot
	tx, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(tx.Hash)

	// TODO: ton.WaitTrace(tx)
	// // Check receipt
	// receipt, err := testutils.WaitMinedWithTxHash(context.Background(), s.ClientA, common.HexToHash(tx.Hash))
	// s.Require().NoError(err, "Failed to mine deployment transaction")
	// s.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
}
