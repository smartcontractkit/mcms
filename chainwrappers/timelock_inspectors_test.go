package chainwrappers

import (
	"encoding/json"
	"fmt"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/chainwrappers/mocks"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/sdk/evm"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	tonsdk "github.com/smartcontractkit/mcms/sdk/ton"
	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestBuildTimelockInspectors(t *testing.T) {
	t.Parallel()

	noSetup := func(*testing.T, *mocks.ChainAccessor) {}

	tests := []struct {
		name          string
		chainMetadata map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata
		setup         func(t *testing.T, access *mocks.ChainAccessor)
		wantTypes     map[mcmsTypes.ChainSelector]any
		wantErr       string
	}{
		{
			name:          "empty input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{},
			setup:         noSetup,
			wantTypes:     map[mcmsTypes.ChainSelector]any{},
		},
		{
			name:  "unknown chain family",
			setup: noSetup,
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				1: {MCMAddress: "0xabc"},
			},
			wantErr: "chain family: chain family not found for selector 1: unknown chain selector 1",
		},
		{
			name: "missing evm client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xevm"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().EVMClient(mock.Anything).Return(nil, false)
			},
			wantErr: "missing EVM chain client",
		},
		{
			name: "missing solana client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector): {MCMAddress: "0xsolana"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().SolanaClient(mock.Anything).Return(nil, false)
			},
			wantErr: "missing Solana chain client",
		},
		{
			name: "missing aptos client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): {MCMAddress: "0xaptos"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().AptosClient(mock.Anything).Return(nil, false)
			},
			wantErr: "missing Aptos chain client",
		},
		{
			name: "aptos invalid metadata",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): {
					MCMAddress:       "0xaptos",
					AdditionalFields: json.RawMessage("{"),
				},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().AptosClient(mock.Anything).Return(nil, true)
			},
			wantErr: "parse aptos metadata",
		},
		{
			name: "missing sui client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
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
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().SuiClient(mock.Anything).Return(nil, false)
			},
			wantErr: "missing Sui chain client",
		},
		{
			name: "missing sui signer",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
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
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().SuiClient(mock.Anything).Return(nil, true)
				access.EXPECT().SuiSigner(mock.Anything).Return(nil, false)
			},
			wantErr: "missing Sui signer",
		},
		{
			name: "sui invalid metadata",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.SUI_TESTNET.Selector): {
					MCMAddress:       "0xsui",
					AdditionalFields: json.RawMessage("{"),
				},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().SuiClient(mock.Anything).Return(nil, true)
				access.EXPECT().SuiSigner(mock.Anything).Return(nil, true)
			},
			wantErr: "parse sui metadata",
		},
		{
			name: "missing ton client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector): {MCMAddress: "0xton"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().TonClient(mock.Anything).Return(nil, false)
			},
			wantErr: "missing TON chain client",
		},
		{
			name: "canton missing participant",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.CANTON_TESTNET.Selector): {MCMAddress: "0xcanton"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().CantonChain(mock.Anything).Return(cantonsdk.Chain{}, true)
			},
			wantErr: "missing Canton chain participant",
		},
		{
			name: "canton missing chain",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.CANTON_TESTNET.Selector): {MCMAddress: "0xcanton"},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().CantonChain(mock.Anything).Return(cantonsdk.Chain{}, false)
			},
			wantErr: "missing Canton chain participant",
		},
		{
			name: "all supported families",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xevm"},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xsolana"},
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector):            {MCMAddress: "0xaptos"},
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector):              {MCMAddress: "0xton"},
				mcmsTypes.ChainSelector(chainsel.CANTON_TESTNET.Selector):           {MCMAddress: "0xcanton"},
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
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()

				access.EXPECT().EVMClient(mock.Anything).Return(nil, true)
				access.EXPECT().SolanaClient(mock.Anything).Return(nil, true)
				access.EXPECT().AptosClient(mock.Anything).Return(nil, true)
				access.EXPECT().TonClient(mock.Anything).Return(nil, true)
				access.EXPECT().SuiClient(mock.Anything).Return(nil, true)
				access.EXPECT().SuiSigner(mock.Anything).Return(nil, true)
				access.EXPECT().CantonChain(mock.Anything).Return(cantonsdk.Chain{
					Participants: []cantonsdk.Participant{
						{PartyID: "party::testnet"},
					},
				}, true)
			},
			wantTypes: map[mcmsTypes.ChainSelector]any{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): (*evm.TimelockInspector)(nil),
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            (*solanasdk.TimelockInspector)(nil),
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector):            (*aptos.TimelockInspector)(nil),
				mcmsTypes.ChainSelector(chainsel.TON_TESTNET.Selector):              (*tonsdk.TimelockInspector)(nil),
				mcmsTypes.ChainSelector(chainsel.CANTON_TESTNET.Selector):           (*cantonsdk.TimelockInspector)(nil),
				mcmsTypes.ChainSelector(chainsel.SUI_TESTNET.Selector):              (*sui.TimelockInspector)(nil),
			},
		},
		{
			name: "aptos curse mcms from metadata",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): {
					MCMAddress: "0xaptos",
					AdditionalFields: json.RawMessage(
						fmt.Sprintf(`{"role":%d,"mcmsType":"%s"}`, aptos.TimelockRoleProposer, aptos.MCMSTypeCurse)),
				},
			},
			setup: func(t *testing.T, access *mocks.ChainAccessor) {
				t.Helper()
				access.EXPECT().AptosClient(mock.Anything).Return(nil, true)
			},
			wantTypes: map[mcmsTypes.ChainSelector]any{
				mcmsTypes.ChainSelector(chainsel.APTOS_TESTNET.Selector): (*aptos.TimelockInspector)(nil),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			access := mocks.NewChainAccessor(t)
			if tc.setup != nil {
				tc.setup(t, access)
			}

			inspectors, err := BuildTimelockInspectors(access, tc.chainMetadata)

			if tc.wantErr == "" {
				require.NoError(t, err)
				require.Len(t, inspectors, len(tc.wantTypes))
				for selector, expectedType := range tc.wantTypes {
					inspector, ok := inspectors[selector]
					require.True(t, ok)
					require.IsType(t, expectedType, inspector)
				}
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
