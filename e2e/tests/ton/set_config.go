//go:build e2e

package tone2e

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/suite"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/hash"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
)

// SetConfigTestSuite tests signing a proposal and converting back to a file
type SetConfigTestSuite struct {
	suite.Suite
	e2e.TestSetup

	wallet   *wallet.Wallet
	mcmsAddr string
}

// SetupSuite runs before the test suite
func (s *SetConfigTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	var err error
	s.wallet, err = LocalWalletDefault(s.TonClient)
	s.Require().NoError(err)

	s.deployMCMSContract()
}

func (s *SetConfigTestSuite) deployMCMSContract() {
	ctx := s.T().Context()

	amount := tlb.MustFromTON("0.3")
	chainID, err := strconv.ParseInt(s.TonBlockchain.ChainID, 10, 64)
	s.Require().NoError(err)
	data := mcms.EmptyDataFrom(hash.CRC32("test.set_config.mcms"), s.wallet.Address(), chainID)
	mcmsAddr, err := DeployMCMSContract(ctx, s.TonClient, s.wallet, amount, data)
	s.Require().NoError(err)
	s.mcmsAddr = mcmsAddr.String()
}

func (s *SetConfigTestSuite) TestSetConfigInspect() {
	// Signers in each group need to be sorted alphabetically
	signers := testutils.MakeNewECDSASigners(30)

	amount := tlb.MustFromTON("0.3")
	configurerTON, err := mcmston.NewConfigurer(s.wallet, amount)
	s.Require().NoError(err)

	inspectorTON := mcmston.NewInspector(s.TonClient)

	tests := []struct {
		name       string
		config     types.Config
		configurer sdk.Configurer
		inspector  sdk.Inspector
		wantErr    error
	}{
		{
			name: "config small/default",
			config: types.Config{
				Quorum:  1,
				Signers: []common.Address{signers[0].Address()},
				GroupSigners: []types.Config{
					{
						Quorum:       1,
						Signers:      []common.Address{signers[1].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
			configurer: configurerTON,
			inspector:  inspectorTON,
		},
		{
			name: "config proposer",
			config: types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[0].Address(),
					signers[1].Address(),
					signers[2].Address(),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 4,
						Signers: []common.Address{
							signers[3].Address(),
							signers[4].Address(),
							signers[5].Address(),
							signers[6].Address(),
							signers[7].Address(),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									signers[8].Address(),
									signers[9].Address(),
								},
								GroupSigners: []types.Config{},
							},
						},
					},
					{
						Quorum: 3,
						Signers: []common.Address{
							signers[10].Address(),
							signers[11].Address(),
							signers[12].Address(),
							signers[13].Address(),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			configurer: configurerTON,
			inspector:  inspectorTON,
		},
		{
			name: "config canceller",
			config: types.Config{
				Quorum: 1,
				Signers: []common.Address{
					signers[14].Address(),
					signers[15].Address(),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							signers[16].Address(),
							signers[17].Address(),
							signers[18].Address(),
							signers[19].Address(),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			configurer: configurerTON,
			inspector:  inspectorTON,
		},
		{
			name: "config proposer",
			config: types.Config{
				Quorum:  2,
				Signers: []common.Address{},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							signers[20].Address(),
							signers[21].Address(),
							signers[22].Address(),
							signers[23].Address(),
						},
						GroupSigners: []types.Config{},
					}, {
						Quorum: 2,
						Signers: []common.Address{
							signers[24].Address(),
							signers[25].Address(),
							signers[26].Address(),
							signers[27].Address(),
						},
						GroupSigners: []types.Config{},
					}, {
						Quorum: 1,
						Signers: []common.Address{
							signers[28].Address(),
							signers[29].Address(),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			configurer: configurerTON,
			inspector:  inspectorTON,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := s.T().Context()
			// Set config
			{
				res, err := tt.configurer.SetConfig(ctx, s.mcmsAddr, &tt.config, true)
				s.Require().NoError(err, "setting config on MCMS contract")

				s.Require().NotNil(res.Hash)
				s.Require().NotNil(res.RawData)

				tx, ok := res.RawData.(*tlb.Transaction)
				s.Require().True(ok)
				s.Require().NotNil(tx.Description)

				err = tracetracking.WaitForTrace(ctx, s.TonClient, tx)
				s.Require().NoError(err)
			}

			{
				gotCount, err := tt.inspector.GetOpCount(ctx, s.mcmsAddr)
				s.Require().NoError(err, "getting config on MCMS contract")
				s.Require().Equal(uint64(0), gotCount)
			}

			// Assert that config has been set
			{
				gotConfig, err := tt.inspector.GetConfig(ctx, s.mcmsAddr)
				s.Require().NoError(err, "getting config on MCMS contract")
				s.Require().NotNil(gotConfig)
				s.Require().Equal(&tt.config, gotConfig)
			}
		})
	}
}
