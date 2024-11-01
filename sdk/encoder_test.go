package sdk

import (
	"testing"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func Test_NewEncoder(t *testing.T) {
	t.Parallel()

	// Static inputs
	var (
		giveTxCount              = uint64(5)
		giveOverridePreviousRoot = false
	)

	tests := []struct {
		name         string
		giveSelector types.ChainSelector
		giveIsSim    bool
		want         Encoder
		wantErr      string
	}{
		{
			name:         "success: returns an EVM encoder (not simulated)",
			giveSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			giveIsSim:    false,
			want: &evm.EVMEncoder{
				TxCount:              giveTxCount,
				ChainID:              cselectors.ETHEREUM_TESTNET_SEPOLIA.EvmChainID,
				OverridePreviousRoot: false,
			},
		},
		{
			name:         "success: returns an EVM encoder (simulated)",
			giveSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			giveIsSim:    true,
			want: &evm.EVMEncoder{
				TxCount:              giveTxCount,
				ChainID:              1337,
				OverridePreviousRoot: false,
			},
		},
		{
			name:         "failure: chain not found for selector",
			giveSelector: types.ChainSelector(0),
			giveIsSim:    true,
			wantErr:      "invalid chain ID: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewEncoder(
				tt.giveSelector,
				giveTxCount,
				giveOverridePreviousRoot,
				tt.giveIsSim,
			)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
