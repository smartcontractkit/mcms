package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cselectors "github.com/smartcontractkit/chain-selectors"
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
			give: ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			want: cselectors.FamilyEVM,
		},
		{
			name: "success: solana",
			give: ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			want: cselectors.FamilySolana,
		},
		{
			name: "success: aptos",
			give: ChainSelector(cselectors.APTOS_TESTNET.Selector),
			want: cselectors.FamilyAptos,
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
