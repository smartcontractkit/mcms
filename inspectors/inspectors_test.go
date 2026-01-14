package inspectors

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestMCMInspectorBuilder_BuildInspectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		chainMetadata           map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata
		chainClientsEVM         map[uint64]sdk.EVMChainClient
		setup                   func(evmClients map[uint64]sdk.EVMChainClient, solanaClients map[uint64]sdk.SolanaChainClient)
		chainClientsSolana      map[uint64]sdk.SolanaChainClient
		expectErr               bool
		errContains             string
		expectedInspectorsCount int
	}{
		{
			name:          "empty input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{},
			chainClientsEVM: map[uint64]sdk.EVMChainClient{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: mocks.NewEVMChainClient(t),
			},
			expectErr: false,
		},
		{
			name: "missing chain client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				1: {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainClientsEVM: map[uint64]sdk.EVMChainClient{},
			expectErr:       true,
			errContains:     "error getting inspector for chain selector 1: error getting chainClient family: chain family not found for selector 1",
		},
		{
			name: "valid input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xabc", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainClientsEVM: map[uint64]sdk.EVMChainClient{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: mocks.NewEVMChainClient(t),
			},
			chainClientsSolana: map[uint64]sdk.SolanaChainClient{
				chainsel.SOLANA_DEVNET.Selector: mocks.NewSolanaChainClient(t),
			},
			expectErr: false,
			setup: func(evmClients map[uint64]sdk.EVMChainClient, solanaClients map[uint64]sdk.SolanaChainClient) {
				evmClients[chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector].(*mocks.EVMChainClient).EXPECT().GetClient().Return(nil).Once()
				solanaClients[chainsel.SOLANA_DEVNET.Selector].(*mocks.SolanaChainClient).EXPECT().GetClient().Return(nil).Once()
			},
			expectedInspectorsCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			allChains := mocks.NewBlockChains(t)

			if tc.expectedInspectorsCount > 0 {
				tc.setup(tc.chainClientsEVM, tc.chainClientsSolana)
				allChains.EXPECT().EVMChains().Return(tc.chainClientsEVM)
				allChains.EXPECT().SolanaChains().Return(tc.chainClientsSolana)
			}

			builder := NewMCMInspectorFetcher(allChains)
			inspectors, err := builder.FetchInspectors(tc.chainMetadata, &mcms.TimelockProposal{})
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
