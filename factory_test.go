package mcms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
)

func TestNewEncoder(t *testing.T) {
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
		want         sdk.Encoder
		wantErr      string
	}{
		{
			name:         "success: returns an EVM encoder (not simulated)",
			giveSelector: chaintest.Chain2Selector,
			giveIsSim:    false,
			want: &evm.Encoder{
				TxCount:              giveTxCount,
				ChainSelector:        chaintest.Chain2Selector,
				OverridePreviousRoot: false,
				IsSim:                false,
			},
		},
		{
			name:         "success: returns an EVM encoder (simulated)",
			giveSelector: chaintest.Chain2Selector,
			giveIsSim:    true,
			want: &evm.Encoder{
				ChainSelector:        chaintest.Chain2Selector,
				TxCount:              giveTxCount,
				OverridePreviousRoot: false,
				IsSim:                true,
			},
		},
		{
			name:         "success: returns a Solana encoder (not simulated)",
			giveSelector: chaintest.Chain4Selector,
			giveIsSim:    false,
			want: &solana.Encoder{
				TxCount:              giveTxCount,
				ChainSelector:        chaintest.Chain4Selector,
				OverridePreviousRoot: false,
			},
		},
		{
			name:         "success: returns an Aptos encoder (not simulated)",
			giveSelector: chaintest.Chain5Selector,
			giveIsSim:    false,
			want: &aptos.Encoder{
				TxCount:              giveTxCount,
				ChainSelector:        chaintest.Chain5Selector,
				OverridePreviousRoot: false,
			},
		},
		{
			name:         "success: returns a Sui encoder",
			giveSelector: chaintest.Chain6Selector,
			giveIsSim:    false,
			want: &sui.Encoder{
				TxCount:              giveTxCount,
				ChainSelector:        chaintest.Chain6Selector,
				OverridePreviousRoot: false,
			},
		},
		{
			name:         "failure: chain not found for selector",
			giveSelector: chaintest.ChainInvalidSelector,
			giveIsSim:    true,
			wantErr:      "chain family not found for selector 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newEncoder(
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

func TestNewTimelockConverter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		chainSelector types.ChainSelector
		want          sdk.TimelockConverter
		wantErr       string
	}{
		{
			name:          "success: EVM executor",
			chainSelector: chaintest.Chain1Selector,
			want:          &evm.TimelockConverter{},
		},
		{
			name:          "success: Solana executor",
			chainSelector: chaintest.Chain4Selector,
			want:          &solana.TimelockConverter{},
		},
		{
			name:          "success: Sui executor",
			chainSelector: chaintest.Chain6Selector,
			want:          &sui.TimelockConverter{},
		},
		{
			name:          "failure: unknown selector",
			chainSelector: types.ChainSelector(123456789),
			wantErr:       "chain family not found for selector 123456789",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newTimelockConverter(tt.chainSelector)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
