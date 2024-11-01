package evm

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

func Test_EVMConfigurator_ToConfig(t *testing.T) {
	t.Parallel()

	var (
		signer1 = common.HexToAddress("0x1")
		signer2 = common.HexToAddress("0x2")
	)

	tests := []struct {
		name    string
		give    bindings.ManyChainMultiSigConfig
		want    *config.Config
		wantErr string
	}{
		{
			name: "success: converts binding config to config",
			give: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{1, 1},
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Group: 0, Index: 0},
					{Addr: signer2, Group: 1, Index: 1},
				},
			},
			want: &config.Config{
				Quorum:  1,
				Signers: []common.Address{signer1},
				GroupSigners: []config.Config{
					{
						Quorum:       1,
						Signers:      []common.Address{signer2},
						GroupSigners: []config.Config{},
					},
				},
			},
		},
		{
			name: "failure: validation error on resulting config",
			give: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{0, 1}, // A zero quorum makes this invalid
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Group: 0, Index: 0},
					{Addr: signer2, Group: 1, Index: 1},
				},
			},
			wantErr: "invalid MCMS config: Quorum must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configurator := EVMConfigurator{}
			got, err := configurator.ToConfig(tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_SetConfigInputs(t *testing.T) {
	t.Parallel()

	var (
		signer1 = common.HexToAddress("0x1")
		signer2 = common.HexToAddress("0x2")
		signer3 = common.HexToAddress("0x3")
		signer4 = common.HexToAddress("0x4")
		signer5 = common.HexToAddress("0x5")
	)

	tests := []struct {
		name       string
		giveConfig config.Config
		want       bindings.ManyChainMultiSigConfig
		wantErr    string
	}{
		{
			name: "success: root signers with some groups",
			giveConfig: config.Config{
				Quorum:  1,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []config.Config{
					{Quorum: 1, Signers: []common.Address{signer3}},
				},
			},
			want: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{1, 1},
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Index: 0, Group: 0},
					{Addr: signer2, Index: 1, Group: 0},
					{Addr: signer3, Index: 2, Group: 1},
				},
			},
		},
		{
			name: "success: root signers with some groups and increased quorum",
			giveConfig: config.Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []config.Config{
					{Quorum: 1, Signers: []common.Address{signer3}},
				},
			},
			want: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{2, 1},
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Index: 0, Group: 0},
					{Addr: signer2, Index: 1, Group: 0},
					{Addr: signer3, Index: 2, Group: 1},
				},
			},
		},
		{
			name: "success: only root signers",
			giveConfig: config.Config{
				Quorum:       1,
				Signers:      []common.Address{signer1, signer2},
				GroupSigners: []config.Config{},
			},
			want: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{1, 0},
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Index: 0, Group: 0},
					{Addr: signer2, Index: 1, Group: 0},
				},
			},
		},
		{
			name: "success: only groups",
			giveConfig: config.Config{
				Quorum:  2,
				Signers: []common.Address{},
				GroupSigners: []config.Config{
					{Quorum: 1, Signers: []common.Address{signer1}},
					{Quorum: 1, Signers: []common.Address{signer2}},
					{Quorum: 1, Signers: []common.Address{signer3}},
				},
			},
			want: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{2, 1, 1, 1},
				GroupParents: [32]uint8{0, 0, 0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Index: 0, Group: 1},
					{Addr: signer2, Index: 1, Group: 2},
					{Addr: signer3, Index: 2, Group: 3},
				},
			},
		},
		{
			name: "success: nested signers and groups",
			giveConfig: config.Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []config.Config{
					{
						Quorum:  1,
						Signers: []common.Address{signer3},
						GroupSigners: []config.Config{
							{Quorum: 1, Signers: []common.Address{signer4}},
						},
					},
					{
						Quorum:  1,
						Signers: []common.Address{signer5},
					},
				},
			},
			want: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{2, 1, 1, 1},
				GroupParents: [32]uint8{0, 0, 1, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Index: 0, Group: 0},
					{Addr: signer2, Index: 1, Group: 0},
					{Addr: signer3, Index: 2, Group: 1},
					{Addr: signer4, Index: 3, Group: 2},
					{Addr: signer5, Index: 4, Group: 3},
				},
			},
		},
		{
			name: "success: unsorted signers and groups",
			giveConfig: config.Config{
				Quorum: 2,
				// Root signers are out of order (signer2 is before signer1)
				Signers: []common.Address{signer2, signer1},
				// Group signers are out of order (signer5 is before the signer4 group)
				GroupSigners: []config.Config{
					{
						Quorum:  1,
						Signers: []common.Address{signer3},
						GroupSigners: []config.Config{
							{Quorum: 1, Signers: []common.Address{signer5}},
						},
					},
					{
						Quorum:  1,
						Signers: []common.Address{signer4},
					},
				},
			},
			want: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{2, 1, 1, 1},
				GroupParents: [32]uint8{0, 0, 1, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: signer1, Index: 0, Group: 0},
					{Addr: signer2, Index: 1, Group: 0},
					{Addr: signer3, Index: 2, Group: 1},
					{Addr: signer4, Index: 3, Group: 3},
					{Addr: signer5, Index: 4, Group: 2},
				},
			},
		},
		{
			name: "failure: signer count cannot exceed 255",
			giveConfig: config.Config{
				Quorum:  1,
				Signers: make([]common.Address, math.MaxUint8+1),
			},
			wantErr: "too many signers: 256 max number is 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configurator := EVMConfigurator{}
			got, err := configurator.SetConfigInputs(tt.giveConfig)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
