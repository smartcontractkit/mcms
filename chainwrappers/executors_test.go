package chainwrappers

import (
	"testing"

	gethbind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	sol "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	tonwallet "github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/mcms/chainwrappers/mocks"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	aptosmocks "github.com/smartcontractkit/mcms/sdk/aptos/mocks/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	suibindmocks "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	suimocks "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	tonmocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

var (
	evmSelector   = mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)
	solSelector   = mcmstypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector)
	aptosSelector = mcmstypes.ChainSelector(chainsel.APTOS_TESTNET.Selector)
	suiSelector   = mcmstypes.ChainSelector(chainsel.SUI_TESTNET.Selector)
	tonSelector   = mcmstypes.ChainSelector(chainsel.TON_TESTNET.Selector)
)

func TestBuildExecutors(t *testing.T) {
	t.Parallel()

	evmClient := evm.ContractDeployBackend(nil) // fixme
	evmSigner := &gethbind.TransactOpts{}
	evmEncoder := evm.NewEncoder(evmSelector, 0, false, false)
	evmExecutor := evm.NewExecutor(evmEncoder, evmClient, &gethbind.TransactOpts{GasLimit: 1234})
	solSigner := &sol.PrivateKey{1, 2, 3}
	solClient := (*solrpc.Client)(nil) // fixme
	solEncoder := solana.NewEncoder(solSelector, 0, false)
	solExecutor := solana.NewExecutor(solEncoder, solClient, *solSigner)
	aptosClient := aptosmocks.NewAptosRpcClient(t)
	aptosSigner := aptosmocks.NewTransactionSigner(t)
	aptosEncoder := aptos.NewEncoder(aptosSelector, 0, false)
	aptosExecutor := aptos.NewExecutor(aptosClient, aptosSigner, aptosEncoder, aptos.TimelockRoleProposer)
	suiClient := suimocks.NewISuiAPI(t)
	suiSigner := suibindmocks.NewSuiSigner(t)
	suiEncoder := sui.NewEncoder(suiSelector, 0, false)
	suiExecutor, err := sui.NewExecutor(suiClient, suiSigner, suiEncoder, nil, "mcms-pkg-id",
		sui.TimelockRoleProposer, "0xsui", "0xaccount123", "0xregistry456", "0xtimelock789")
	require.NoError(t, err)
	tonClient := tonmocks.NewAPIClientWrapped(t)
	tonSigner := &tonwallet.Wallet{}
	tonEncoder := ton.NewEncoder(tonSelector, 0, false)
	tonExecOpts := ton.ExecutorOpts{Encoder: tonEncoder, Client: tonClient, Wallet: tonSigner, Amount: ton.DefaultSendAmount}
	tonExecutor, err := ton.NewExecutor(tonExecOpts)
	require.NoError(t, err)

	tests := []struct {
		name          string
		encoders      map[mcmstypes.ChainSelector]mcmssdk.Encoder
		chainMetadata map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata
		setup         func(accessor *mocks.ChainAccessor)
		want          map[mcmstypes.ChainSelector]mcmssdk.Executor
		wantErr       string
	}{
		{
			name: "success",
			encoders: map[mcmstypes.ChainSelector]mcmssdk.Encoder{
				evmSelector:   evmEncoder,
				solSelector:   solEncoder,
				aptosSelector: aptosEncoder,
				suiSelector:   suiEncoder,
				tonSelector:   tonEncoder,
			},
			chainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {
					MCMAddress:       "0xevm",
					StartingOpCount:  0,
					AdditionalFields: []byte(`{"gasLimit": 1234}`),
				},
				mcmstypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector): {
					MCMAddress:      "0xsolana",
					StartingOpCount: 0,
				},
				mcmstypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): {
					MCMAddress:      "0xaptos",
					StartingOpCount: 0,
				},
				mcmstypes.ChainSelector(chainsel.SUI_TESTNET.Selector): {
					MCMAddress:      "0xsui",
					StartingOpCount: 0,
					AdditionalFields: []byte(`{
						"role":0,
						"mcms_package_id":"mcms-pkg-id",
						"account_obj":"0xaccount123",
						"registry_obj":"0xregistry456",
						"timelock_obj":"0xtimelock789",
						"deployer_state_obj":"0xdeployer"
					}`),
				},
				mcmstypes.ChainSelector(chainsel.TON_TESTNET.Selector): {
					MCMAddress:      "0xton",
					StartingOpCount: 0,
				},
			},
			setup: func(accessor *mocks.ChainAccessor) {
				accessor.EXPECT().EVMClient(mock.Anything).Return(nil, true)
				accessor.EXPECT().EVMSigner(mock.Anything).Return(evmSigner, true)
				accessor.EXPECT().SolanaClient(mock.Anything).Return(nil, true)
				accessor.EXPECT().SolanaSigner(mock.Anything).Return(solSigner, true)
				accessor.EXPECT().AptosClient(mock.Anything).Return(nil, true)
				accessor.EXPECT().AptosSigner(mock.Anything).Return(nil, true)
				accessor.EXPECT().SuiClient(mock.Anything).Return(nil, true)
				accessor.EXPECT().SuiSigner(mock.Anything).Return(nil, true)
				accessor.EXPECT().TonClient(mock.Anything).Return(tonClient, true)
				accessor.EXPECT().TonSigner(mock.Anything).Return(tonSigner, true)
			},
			want: map[mcmstypes.ChainSelector]mcmssdk.Executor{
				evmSelector:   evmExecutor,
				solSelector:   solExecutor,
				aptosSelector: aptosExecutor,
				suiSelector:   suiExecutor,
				tonSelector:   tonExecutor,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chainAccessor := mocks.NewChainAccessor(t)
			tt.setup(chainAccessor)

			got, err := BuildExecutors(chainAccessor, tt.chainMetadata, tt.encoders, mcmstypes.TimelockActionSchedule)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
