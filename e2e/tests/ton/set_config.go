//go:build e2e

package tone2e

import (
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/suite"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"
)

func makeRandomTestWallet(api wallet.TonAPI, networkGlobalID int32) (*wallet.Wallet, error) {
	v5r1Config := wallet.ConfigV5R1Final{
		NetworkGlobalID: networkGlobalID,
		Workchain:       0,
	}
	return wallet.FromSeed(api, wallet.NewSeed(), v5r1Config)
}

// SetConfigTestSuite tests signing a proposal and converting back to a file
type SetConfigTestSuite struct {
	suite.Suite
	e2e.TestSetup

	wallet   *wallet.Wallet
	mcmsAddr string
}

// SetupSuite runs before the test suite
func (t *SetConfigTestSuite) SetupSuite() {
	t.TestSetup = *e2e.InitializeSharedTestSetup(t.T())

	walletVersion := wallet.HighloadV2Verified //nolint:staticcheck // only option in mylocalton-docker

	var err error
	t.wallet, err = wallet.FromSeed(t.TonClient, strings.Fields(blockchain.DefaultTonHlWalletMnemonic), walletVersion)
	t.Require().NoError(err)

	t.deployMCMSContract()
}

func (t *SetConfigTestSuite) deployMCMSContract() {
	amount := tlb.MustFromTON("10")
	msgBody := cell.BeginCell().EndCell()      // empty cell
	var contractCode *cell.Cell                // TODO: load contract code
	contractData := cell.BeginCell().EndCell() // TODO: replace empty cell with init storage
	workchain := int8(0)

	addr, tx, _, err := t.wallet.DeployContractWaitTransaction(t.T().Context(), amount, msgBody, contractCode, contractData, workchain)
	t.Require().NoError(err)
	t.Require().NotNil(tx)

	t.mcmsAddr = addr.String()
}

func (t *SetConfigTestSuite) Test_TON_SetConfigInspect() {
	// Signers in each group need to be sorted alphabetically
	signers := [30]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return strings.Compare(strings.ToLower(a.Hex()), strings.ToLower(b.Hex()))
	})

	// TODO: use from test suite
	var wallet *wallet.Wallet
	amount := tlb.MustFromTON("0")
	configurerTON, err := tonmcms.NewConfigurer(wallet, amount)
	t.Require().NoError(err)

	inspectorTON := tonmcms.NewInspector(t.TonClient, tonmcms.NewConfigTransformer())
	t.Require().NoError(err)

	tests := []struct {
		name       string
		config     types.Config
		configurer sdk.Configurer
		inspector  sdk.Inspector
		wantErr    error
	}{
		{
			name: "config proposer",
			config: types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[0],
					signers[1],
					signers[2],
				},
				GroupSigners: []types.Config{
					{
						Quorum: 4,
						Signers: []common.Address{
							signers[3],
							signers[4],
							signers[5],
							signers[6],
							signers[7],
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									signers[8],
									signers[9],
								},
								GroupSigners: []types.Config{},
							},
						},
					},
					{
						Quorum: 3,
						Signers: []common.Address{
							signers[10],
							signers[11],
							signers[12],
							signers[13],
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
					signers[14],
					signers[15],
				},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							signers[16],
							signers[17],
							signers[18],
							signers[19],
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
							signers[20],
							signers[21],
							signers[22],
							signers[23],
						},
						GroupSigners: []types.Config{},
					}, {
						Quorum: 2,
						Signers: []common.Address{
							signers[24],
							signers[25],
							signers[26],
							signers[27],
						},
						GroupSigners: []types.Config{},
					}, {
						Quorum: 1,
						Signers: []common.Address{
							signers[28],
							signers[29],
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
		t.Run(tt.name, func() {
			// Set config
			{
				res, err := tt.configurer.SetConfig(t.T().Context(), t.mcmsAddr, &tt.config, true)
				t.Require().NoError(err, "setting config on Aptos mcms contract")

				// TODO: wait for tx, verify success
				t.Require().NotNil(res.Hash)
				t.Require().NotNil(res.RawData)
				tx, ok := res.RawData.(*tlb.Transaction)
				t.Require().True(ok)
				t.Require().NotNil(tx.Description)
			}

			// Assert that config has been set
			{
				gotConfig, err := tt.inspector.GetConfig(t.T().Context(), t.mcmsAddr)
				t.Require().NoError(err, "getting config on Aptos mcms contract")
				t.Require().NotNil(gotConfig)
				t.Require().Equal(tt.config, gotConfig)
			}
		})
	}
}
