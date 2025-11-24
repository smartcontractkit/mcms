//go:build e2e

package tone2e

import (
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	commonton "github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"
)

const (
	EnvPathContracts = "PATH_CONTRACTS_TON"

	PathContractsMCMS     = "mcms.MCMS.compiled.json"
	PathContractsTimelock = "mcms.RBACTimelock.compiled.json"
)

// TODO: duplicated utils with unit tests [START]

func must[E any](out E, err error) E {
	if err != nil {
		panic(err)
	}
	return out
}

// TODO: duplicated utils with unit tests [END]

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
		Signers: must(tvm.MakeDict(map[*big.Int]mcms.Signer{}, tvm.KeyUINT256)),
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
	contract, _, err := wrappers.Deploy(&client, contractCode, contractData, amount, msgBody)
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
	signers := [30]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return strings.Compare(strings.ToLower(a.Hex()), strings.ToLower(b.Hex()))
	})

	amount := tlb.MustFromTON("0.3")
	configurerTON, err := tonmcms.NewConfigurer(t.wallet, amount)
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
