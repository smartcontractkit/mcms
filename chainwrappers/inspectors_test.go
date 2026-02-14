package chainwrappers

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/chainwrappers/mocks"

	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestMCMInspectorBuilder_BuildInspectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		chainMetadata           map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata
		chainAccess             *mocks.ChainAccessor
		setup                   func(access *mocks.ChainAccessor)
		expectErr               bool
		errContains             string
		expectedInspectorsCount int
	}{
		{
			name:          "empty input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{},
			chainAccess:   mocks.NewChainAccessor(t),
			expectErr:     false,
		},
		{
			name: "missing chain client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				1: {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainAccess: mocks.NewChainAccessor(t),
			expectErr:   true,
			errContains: "error getting chain family: chain family not found for selector 1",
		},
		{
			name: "valid input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xevm", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xsolana", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector):            {MCMAddress: "0xaptos", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector):              {MCMAddress: "0xton", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.SUI_TESTNET.Selector): {
					MCMAddress:      "0xsui",
					StartingOpCount: 0,
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
			chainAccess: mocks.NewChainAccessor(t),
			expectErr:   false,
			setup: func(access *mocks.ChainAccessor) {
				access.EXPECT().EVMClient(mock.Anything).Return(nil, true)
				access.EXPECT().SolanaClient(mock.Anything).Return(nil, true)
				access.EXPECT().AptosClient(mock.Anything).Return(nil, true)
				access.EXPECT().SuiClient(mock.Anything).Return(nil, nil, true)
				access.EXPECT().TonClient(mock.Anything).Return(nil, true)
			},
			expectedInspectorsCount: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.chainAccess = mocks.NewChainAccessor(t)
			if tc.expectedInspectorsCount > 0 {
				tc.setup(tc.chainAccess)
			}

			inspectors, err := BuildInspectors(tc.chainAccess, tc.chainMetadata, mcmsTypes.TimelockActionSchedule)
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				require.Len(t, inspectors, tc.expectedInspectorsCount)
			}
		})
	}
}
