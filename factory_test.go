package mcms

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
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
		want         sdk.Encoder
		wantErr      string
	}{
		{
			name:         "success: returns an EVM encoder (not simulated)",
			giveSelector: chaintest.Chain2Selector,
			giveIsSim:    false,
			want: &evmsdk.Encoder{
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
			want: &evmsdk.Encoder{
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
			want: &solanasdk.Encoder{
				TxCount:              giveTxCount,
				ChainSelector:        chaintest.Chain4Selector,
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

func Test_newTimelockConverterFromExecutor(t *testing.T) {
	t.Parallel()

	solanaClient := &rpc.Client{}
	solanaAuth, _ := solana.NewRandomPrivateKey()

	tests := []struct {
		name          string
		chainSelector types.ChainSelector
		executor      sdk.TimelockExecutor
		want          sdk.TimelockConverter
		wantErr       string
	}{
		{
			name:          "success: EVM executor",
			chainSelector: chaintest.Chain1Selector,
			executor:      &evmsdk.TimelockExecutor{},
			want:          &evmsdk.TimelockConverter{},
		},
		{
			name:          "success: Solana executor",
			chainSelector: chaintest.Chain4Selector,
			executor:      solanasdk.NewTimelockExecutor(solanaClient, solanaAuth),
			want:          solanasdk.NewTimelockConverter(solanaClient, solanaAuth.PublicKey()),
		},
		{
			name:          "failure: unknown selector",
			chainSelector: types.ChainSelector(123456789),
			executor:      &UnknownTimelockExecutor{},
			wantErr:       "chain family not found for selector 123456789",
		},
		{
			name:          "failure: unknown executor type",
			chainSelector: chaintest.Chain4Selector,
			executor:      &UnknownTimelockExecutor{},
			wantErr:       "unable to cast sdk executor to solana TimelockExecutor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newTimelockConverterFromExecutor(tt.chainSelector, tt.executor)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

type UnknownTimelockExecutor struct {
	sdk.TimelockExecutor
}
