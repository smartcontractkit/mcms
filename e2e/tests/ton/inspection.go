//go:build e2e

package tone2e

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/hash"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/internal/testutils"
	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// InspectionTestSuite defines the test suite
type InspectionTestSuite struct {
	suite.Suite
	e2e.TestSetup

	wallet   *wallet.Wallet
	mcmsAddr string

	signers []testutils.ECDSASigner
}

// SetupSuite runs before the test suite
func (s *InspectionTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	// Generate few test signers
	s.signers = testutils.MakeNewECDSASigners(2)

	var err error
	s.wallet, err = LocalWalletDefault(s.TonClient)
	s.Require().NoError(err)

	s.deployMCMSContract()
}

func (s *InspectionTestSuite) deployMCMSContract() {
	ctx := s.T().Context()

	amount := tlb.MustFromTON("0.3")
	chainID, err := strconv.ParseInt(s.TonBlockchain.ChainID, 10, 64)
	s.Require().NoError(err)
	data := MCMSEmptyDataFrom(hash.CRC32("test.inspection.mcms"), s.wallet.Address(), chainID)
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
func (s *InspectionTestSuite) TestGetConfig() {
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

// TestGetOpCount checks contract operation count
func (s *InspectionTestSuite) TestGetOpCount() {
	ctx := s.T().Context()

	inspector := mcmston.NewInspector(s.TonClient)
	opCount, err := inspector.GetOpCount(ctx, s.mcmsAddr)

	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count does not match")
}

// TestGetRoot checks contract root
func (s *InspectionTestSuite) TestGetRoot() {
	ctx := s.T().Context()

	inspector := mcmston.NewInspector(s.TonClient)
	root, validUntil, err := inspector.GetRoot(ctx, s.mcmsAddr)

	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(common.Hash{}, root, "Roots do not match")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil does not match")
}

// TestGetRootMetadata checks contract root metadata
func (s *InspectionTestSuite) TestGetRootMetadata() {
	ctx := s.T().Context()

	inspector := mcmston.NewInspector(s.TonClient)
	metadata, err := inspector.GetRootMetadata(ctx, s.mcmsAddr)

	s.Require().NoError(err, "Failed to get root metadata from contract")
	s.Require().Equal(metadata.MCMAddress, s.mcmsAddr, "MCMAddress does not match")
	s.Require().Equal(uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
