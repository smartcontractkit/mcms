package types //nolint:revive,nolintlint // allow pkg name 'types'

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

func TestGetChainSelectorFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    ChainSelector
		want    string
		wantErr string
	}{
		{
			name: "success: evm",
			give: ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
			want: chainsel.FamilyEVM,
		},
		{
			name: "success: solana",
			give: ChainSelector(chainsel.SOLANA_DEVNET.Selector),
			want: chainsel.FamilySolana,
		},
		{
			name: "success: aptos",
			give: ChainSelector(chainsel.APTOS_TESTNET.Selector),
			want: chainsel.FamilyAptos,
		},
		{
			name:    "invalid chain selector",
			give:    0,
			wantErr: "chain family not found for selector 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetChainSelectorFamily(tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
