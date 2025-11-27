//go:build e2e

package tone2e

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/suite"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/wrappers"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	commonton "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
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
func (t *SetConfigTestSuite) SetupSuite() {
	t.TestSetup = *e2e.InitializeSharedTestSetup(t.T())

	walletVersion := wallet.HighloadV2Verified //nolint:staticcheck // only option in mylocalton-docker
	mcWallet, err := wallet.FromSeed(t.TonClient, strings.Fields(blockchain.DefaultTonHlWalletMnemonic), walletVersion)
	t.Require().NoError(err)

	time.Sleep(8 * time.Second)

	mcFunderWallet, err := wallet.FromPrivateKeyWithOptions(t.TonClient, mcWallet.PrivateKey(), walletVersion, wallet.WithWorkchain(-1))
	t.Require().NoError(err)

	// subwallet 42 has balance
	t.wallet, err = mcFunderWallet.GetSubwallet(uint32(42))
	t.Require().NoError(err)

	t.deployMCMSContract()
}

func (t *SetConfigTestSuite) deployMCMSContract() {
	amount := tlb.MustFromTON("0.05")
	msgBody := cell.BeginCell().EndCell() // empty cell, top up

	contractPath := filepath.Join(os.Getenv(EnvPathContracts), PathContractsMCMS)
	contractCode, err := wrappers.ParseCompiledContract(contractPath)
	t.Require().NoError(err)

	contractData, err := tlb.ToCell(mcms.Data{
		ID: 4,
		Ownable: commonton.Ownable2Step{
			Owner:        t.wallet.Address(),
			PendingOwner: nil,
		},
		Oracle:  tvm.ZeroAddress,
		Signers: must(tvm.MakeDict(map[*big.Int]mcms.Signer{}, 160)), // TODO: tvm.KeyUINT160
		Config: mcms.Config{
			Signers:      must(tvm.MakeDictFrom([]mcms.Signer{}, tvm.KeyUINT8)),
			GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{}, tvm.KeyUINT8)),
			GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{}, tvm.KeyUINT8)),
		},
		SeenSignedHashes: must(tvm.MakeDict(map[*big.Int]mcms.SeenSignedHash{}, tvm.KeyUINT256)),
		RootInfo: mcms.RootInfo{
			ExpiringRootAndOpCount: mcms.ExpiringRootAndOpCount{
				Root:       big.NewInt(0),
				ValidUntil: 0,
				OpCount:    17,
				OpPendingInfo: mcms.OpPendingInfo{
					ValidAfter:             0,
					OpFinalizationTimeout:  0,
					OpPendingReceiver:      tvm.ZeroAddress,
					OpPendingBodyTruncated: big.NewInt(0),
				},
			},
			RootMetadata: mcms.RootMetadata{
				ChainID:              big.NewInt(-217),
				MultiSig:             tvm.ZeroAddress,
				PreOpCount:           17,
				PostOpCount:          17,
				OverridePreviousRoot: false,
			},
		},
	})
	t.Require().NoError(err)

	// TODO: extract .WaitTrace(tx) functionality and use here instead of wrapper
	client := tracetracking.NewSignedAPIClient(t.TonClient, *t.wallet)
	contract, _, err := wrappers.Deploy(t.T().Context(), &client, contractCode, contractData, amount, msgBody)
	t.Require().NoError(err)
	addr := contract.Address

	// workchain := int8(-1)
	// addr, tx, _, err := t.wallet.DeployContractWaitTransaction(t.T().Context(), amount, msgBody, contractCode, contractData, workchain)
	t.Require().NoError(err)
	// t.Require().NotNil(tx)

	t.mcmsAddr = addr.String()
}

func (t *SetConfigTestSuite) Test_TON_SetConfigInspect() {
	// Signers in each group need to be sorted alphabetically
	signers := testutils.MakeNewECDSASigners(30)

	amount := tlb.MustFromTON("0.3")
	configurerTON, err := mcmston.NewConfigurer(t.wallet, amount)
	t.Require().NoError(err)

	inspectorTON := mcmston.NewInspector(t.TonClient, mcmston.NewConfigTransformer())
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
		t.Run(tt.name, func() {
			// Set config
			{
				res, err := tt.configurer.SetConfig(t.T().Context(), t.mcmsAddr, &tt.config, true)
				t.Require().NoError(err, "setting config on MCMS contract")

				t.Require().NotNil(res.Hash)
				t.Require().NotNil(res.RawData)

				tx, ok := res.RawData.(*tlb.Transaction)
				t.Require().True(ok)
				t.Require().NotNil(tx.Description)

				// TODO: wait for tx, verify success
				// TODO: implement waiting for tx trace
				// client := tracetracking.NewSignedAPIClient(c.client, *c.wallet)
				// rm, err := client.SendAndWaitForTrace(ctx, *dstAddr, msg)
				time.Sleep(3 * time.Second)
			}

			{
				gotCount, err := tt.inspector.GetOpCount(t.T().Context(), t.mcmsAddr)
				t.Require().NoError(err, "getting config on MCMS contract")
				t.Require().Equal(uint64(17), gotCount)
			}

			// Assert that config has been set
			{
				gotConfig, err := tt.inspector.GetConfig(t.T().Context(), t.mcmsAddr)
				t.Require().NoError(err, "getting config on MCMS contract")
				t.Require().NotNil(gotConfig)
				t.Require().Equal(&tt.config, gotConfig)
			}
		})
	}
}
