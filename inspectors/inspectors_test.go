package inspectors

import (
	"testing"

	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestMCMInspectorBuilder_BuildInspectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		chainMetadata           map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata
		chainAccess             sdk.ChainAccess
		expectErr               bool
		errContains             string
		expectedInspectorsCount int
	}{
		{
			name:          "empty input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{},
			chainAccess:   &fakeChainAccess{},
			expectErr:     false,
		},
		{
			name: "missing chain client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				1: {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainAccess: &fakeChainAccess{},
			expectErr:   true,
			errContains: "error getting chain family: chain family not found for selector 1",
		},
		{
			name: "valid input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xabc", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainAccess: &fakeChainAccess{
				selectors: []uint64{
					chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
					chainsel.SOLANA_DEVNET.Selector,
				},
				evmClients: map[uint64]sdk.ContractDeployBackend{
					chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: nil,
				},
				solanaClients: map[uint64]*solrpc.Client{
					chainsel.SOLANA_DEVNET.Selector: nil,
				},
			},
			expectErr:               false,
			expectedInspectorsCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.chainAccess == nil {
				tc.chainAccess = &fakeChainAccess{}
			}

			builder := NewMCMInspectorFetcher(tc.chainAccess)
			inspectors, err := builder.FetchInspectors(tc.chainMetadata, mcmsTypes.TimelockActionSchedule)
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

type fakeChainAccess struct {
	selectors     []uint64
	evmClients    map[uint64]sdk.ContractDeployBackend
	solanaClients map[uint64]*solrpc.Client
	aptosClients  map[uint64]aptoslib.AptosRpcClient
	suiClients    map[uint64]struct {
		client sui.ISuiAPI
		signer sdk.SuiSigner
	}
}

var _ sdk.ChainAccess = (*fakeChainAccess)(nil)

func (f *fakeChainAccess) Selectors() []uint64 {
	if f == nil {
		return nil
	}

	return f.selectors
}

func (f *fakeChainAccess) EVMClient(selector uint64) (sdk.ContractDeployBackend, bool) {
	if f == nil || f.evmClients == nil {
		return nil, false
	}
	client, ok := f.evmClients[selector]

	return client, ok
}

func (f *fakeChainAccess) SolanaClient(selector uint64) (*solrpc.Client, bool) {
	if f == nil || f.solanaClients == nil {
		return nil, false
	}
	client, ok := f.solanaClients[selector]

	return client, ok
}

func (f *fakeChainAccess) AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool) {
	if f == nil || f.aptosClients == nil {
		var zero aptoslib.AptosRpcClient
		return zero, false
	}
	client, ok := f.aptosClients[selector]

	return client, ok
}

func (f *fakeChainAccess) Sui(selector uint64) (sui.ISuiAPI, sdk.SuiSigner, bool) {
	if f == nil || f.suiClients == nil {
		return nil, nil, false
	}
	client, ok := f.suiClients[selector]
	if !ok {
		return nil, nil, false
	}

	return client.client, client.signer, true
}
