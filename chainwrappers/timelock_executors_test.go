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

func TestBuildTimelockExecutors(t *testing.T) {
	t.Parallel()

	evmClient := evm.ContractDeployBackend(nil) // fixme
	evmSigner := &gethbind.TransactOpts{}
	evmExecutor := evm.NewTimelockExecutor(evmClient, &gethbind.TransactOpts{GasLimit: 1234})
	solSigner := &sol.PrivateKey{1, 2, 3}
	solClient := (*solrpc.Client)(nil) // fixme
	solExecutor := solana.NewTimelockExecutor(solClient, *solSigner)
	aptosClient := aptosmocks.NewAptosRpcClient(t)
	aptosSigner := aptosmocks.NewTransactionSigner(t)
	aptosExecutor := aptos.NewTimelockExecutor(aptosClient, aptosSigner)
	suiClient := suimocks.NewISuiAPI(t)
	suiSigner := suibindmocks.NewSuiSigner(t)
	suiExecutor, err := sui.NewTimelockExecutor(suiClient, suiSigner, nil, "mcms-pkg-id", "0xregistry456", "0xaccount123")
	require.NoError(t, err)
	tonClient := tonmocks.NewAPIClientWrapped(t)
	tonSigner := &tonwallet.Wallet{}
	tonExecutor, err := ton.NewTimelockExecutor(
		ton.TimelockExecutorOpts{Client: tonClient, Wallet: tonSigner, Amount: ton.DefaultSendAmount})
	require.NoError(t, err)

	tests := []struct {
		name          string
		chainMetadata map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata
		setup         func(accessor *mocks.ChainAccessor)
		want          map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor
		wantErr       string
	}{
		{
			name: "success",
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
			want: map[mcmstypes.ChainSelector]mcmssdk.TimelockExecutor{
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

			got, err := BuildTimelockExecutors(chainAccessor, tt.chainMetadata, mcmstypes.TimelockActionSchedule)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
