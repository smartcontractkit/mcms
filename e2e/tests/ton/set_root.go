//go:build e2e

package tone2e

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/hash"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

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
	AddrTimelock = "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8" // static mock address
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
	var client *ton.APIClient
	s.accounts = []*address.Address{
		must(tvm.NewRandomTestWallet(client, chainID)).Address(),
		must(tvm.NewRandomTestWallet(client, chainID)).Address(),
	}

	var err error
	s.wallet, err = LocalWalletDefault(s.TonClient)
	s.Require().NoError(err)

	s.deployMCMSContract()

	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.TonBlockchain.ChainID, s.TonBlockchain.Family)
	s.Require().NoError(err)
	s.chainSelector = types.ChainSelector(chainDetails.ChainSelector)
}

func (s *SetRootTestSuite) deployMCMSContract() {
	ctx := s.T().Context()

	amount := tlb.MustFromTON("0.3")
	chainID, err := strconv.ParseInt(s.TonBlockchain.ChainID, 10, 64)
	s.Require().NoError(err)
	data := MCMSEmptyDataFrom(hash.CRC32("test.set_root.mcms"), s.wallet.Address(), chainID)
	mcmsAddr, err := DeployMCMSContract(ctx, s.TonClient, s.wallet, amount, data)
	s.Require().NoError(err)
	s.mcmsAddr = mcmsAddr.String()

	// Set configuration
	configurerTON, err := mcmston.NewConfigurer(s.wallet, amount)
	s.Require().NoError(err)

	config := &types.Config{
		Quorum:  1,
		Signers: []common.Address{s.signers[0].Address()},
		GroupSigners: []types.Config{
			{
				Quorum:       1,
				Signers:      []common.Address{s.signers[1].Address()},
				GroupSigners: []types.Config{},
			},
		},
	}

	clearRoot := true
	res, err := configurerTON.SetConfig(ctx, s.mcmsAddr, config, clearRoot)
	s.Require().NoError(err, "Failed to set contract configuration")
	s.Require().NotNil(res)

	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx.Description)

	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
}

// TestGetConfig checks contract configuration
func (s *SetRootTestSuite) TestGetConfig() {
	ctx := s.T().Context()

	inspector := mcmston.NewInspector(s.TonClient)
	config, err := inspector.GetConfig(ctx, s.mcmsAddr)
	s.Require().NoError(err, "Failed to get contract configuration")
	s.Require().NotNil(config, "Contract configuration is nil")

	// Check first group
	s.Require().Equal(uint8(1), config.Quorum, "Quorum does not match")
	s.Require().Equal(s.signers[0].Address(), config.Signers[0], "Signers do not match")

	// Check second group
	s.Require().Equal(uint8(1), config.GroupSigners[0].Quorum, "Group quorum does not match")
	s.Require().Equal(s.signers[1].Address(), config.GroupSigners[0].Signers[0], "Group signers do not match")
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
		s.chainSelector: mcmston.NewInspector(s.TonClient),
	}
	signable, err := mcmslib.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcmslib.NewPrivateKeySigner(s.signers[0].Key))
	s.Require().NoError(err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

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
	res, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
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
			s.chainSelector: AddrTimelock,
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
		s.chainSelector: mcmston.NewInspector(s.TonClient),
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

	// Notice: no simulation on TON (like on EVM)

	// Create the chain MCMS proposal executor
	executable, err := mcmslib.NewExecutable(&proposal, executorsMap)
	s.Require().NoError(err)
	// Call SetRoot
	res, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(res.Hash)

	tx, ok := res.RawData.(*tlb.Transaction)
	s.Require().True(ok)
	s.Require().NotNil(tx)

	// Wait and check success
	err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
	s.Require().NoError(err)
}
