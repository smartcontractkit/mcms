package chainwrappers

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gagliardetto/solana-go"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/mcms/chainwrappers/mocks"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestBuildTimelockConfigurers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		chainMetadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata
		setup         func(t *testing.T, access *mocks.ChainAccessor, metadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata)
		expectErr     bool
		errContains   string
		expectTypes   map[mcmsTypes.ChainSelector]any
	}{
		{
			name:          "empty input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{},
			expectErr:     false,
		},
		{
			name: "unknown chain family",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				1: {MCMAddress: "0xabc"},
			},
			expectErr:   true,
			errContains: "error getting chain family",
		},
		{
			name: "all supported families",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xevm"},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xsolana"},
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector):            {MCMAddress: "0xaptos"},
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector):              {MCMAddress: "0xton"},
				mcmsTypes.ChainSelector(chainsel.SUI_TESTNET.Selector): {
					MCMAddress: "0xsui",
					AdditionalFields: []byte(`{
						"role":0,
						"mcms_package_id":"0x123456789abcdef",
						"account_obj":"0xaccount123",
						"registry_obj":"0xregistry456",
						"timelock_obj":"0xtimelock789",
						"deployer_state_obj":"0xdeployer"
					}`),
				},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor, metadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata) {
				t.Helper()

				access.EXPECT().EVMClient(mock.Anything).Return(nil, true)
				access.EXPECT().EVMSigner(mock.Anything).Return(&bind.TransactOpts{}, true)

				access.EXPECT().SolanaClient(mock.Anything).Return(nil, true)
				solKey, err := solana.NewRandomPrivateKey()
				require.NoError(t, err)
				access.EXPECT().SolanaSigner(mock.Anything).Return(&solKey, true)

				access.EXPECT().AptosClient(mock.Anything).Return(nil, true)

				access.EXPECT().TonSigner(mock.Anything).Return(&wallet.Wallet{}, true)
			},
			expectTypes: map[mcmsTypes.ChainSelector]any{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): (*evm.TimelockConfigurer)(nil),
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            (*solanasdk.TimelockConfigurer)(nil),
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector):            (*aptos.TimelockConfigurer)(nil),
				mcmsTypes.ChainSelector(chainsel.SUI_TESTNET.Selector):              (*sui.TimelockConfigurer)(nil),
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector):              (*ton.TimelockConfigurer)(nil),
			},
		},
		{
			name: "aptos curse mcms from metadata",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): {
					MCMAddress: "0xaptos",
				},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor, metadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata) {
				t.Helper()

				b, err := json.Marshal(aptos.AdditionalFieldsMetadata{
					Role:     aptos.TimelockRoleProposer,
					MCMSType: aptos.MCMSTypeCurse,
				})
				require.NoError(t, err)

				entry := metadata[mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector)]
				entry.AdditionalFields = b
				metadata[mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector)] = entry

				access.EXPECT().AptosClient(mock.Anything).Return(nil, true)
			},
			expectTypes: map[mcmsTypes.ChainSelector]any{
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): (*aptos.TimelockConfigurer)(nil),
			},
		},
		{
			name: "missing evm client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xevm"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor, metadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata) {
				t.Helper()

				access.EXPECT().EVMClient(mock.Anything).Return(nil, false)
			},
			expectErr:   true,
			errContains: "missing EVM chain client",
		},
		{
			name: "missing evm signer",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xevm"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor, metadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata) {
				t.Helper()

				access.EXPECT().EVMClient(mock.Anything).Return(nil, true)
				access.EXPECT().EVMSigner(mock.Anything).Return(nil, false)
			},
			expectErr:   true,
			errContains: "missing EVM chain signer",
		},
		{
			name: "missing ton signer",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector): {MCMAddress: "0xton"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor, metadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata) {
				t.Helper()

				access.EXPECT().TonSigner(mock.Anything).Return(nil, false)
			},
			expectErr:   true,
			errContains: "missing TON chain wallet",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			access := mocks.NewChainAccessor(t)
			if tc.setup != nil {
				tc.setup(t, access, tc.chainMetadata)
			}

			configurers, err := BuildTimelockConfigurers(access, tc.chainMetadata, mcmsTypes.TimelockActionSchedule)
			if tc.expectErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)

				return
			}

			require.NoError(t, err)
			require.Len(t, configurers, len(tc.expectTypes))
			for selector, expectedType := range tc.expectTypes {
				configurer, ok := configurers[selector]
				require.True(t, ok)
				require.IsType(t, expectedType, configurer)
			}
		})
	}
}

func TestBuildTimelockConfigurer_NilChainAccess(t *testing.T) {
	t.Parallel()

	_, err := BuildTimelockConfigurer(
		nil,
		mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
		mcmsTypes.TimelockActionSchedule,
		mcmsTypes.ChainMetadata{},
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "chain access is required")
}

func TestBuildTimelockConfigurer_AptosCurseMetadataEncodesCursePackage(t *testing.T) {
	t.Parallel()

	access := mocks.NewChainAccessor(t)
	access.EXPECT().AptosClient(mock.Anything).Return(nil, true)

	metadata := mcmsTypes.ChainMetadata{
		MCMAddress: "0xaptos",
	}
	b, err := json.Marshal(aptos.AdditionalFieldsMetadata{
		Role:     aptos.TimelockRoleProposer,
		MCMSType: aptos.MCMSTypeCurse,
	})
	require.NoError(t, err)
	metadata.AdditionalFields = b

	configurer, err := BuildTimelockConfigurer(
		access,
		mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector),
		mcmsTypes.TimelockActionSchedule,
		metadata,
	)
	require.NoError(t, err)

	result, err := configurer.UpdateDelay(
		t.Context(),
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		3600,
	)
	require.NoError(t, err)
	require.Empty(t, result.Hash)

	tx, ok := result.RawData.(mcmsTypes.Transaction)
	require.True(t, ok)

	var fields aptos.AdditionalFields
	require.NoError(t, json.Unmarshal(tx.AdditionalFields, &fields))
	require.Equal(t, "curse_mcms", fields.PackageName)
	require.Equal(t, "curse_mcms", fields.ModuleName)
	require.Equal(t, "timelock_update_min_delay", fields.Function)
}
