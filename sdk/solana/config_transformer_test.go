package solana

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func Test_ConfigTransformer_ToConfig(t *testing.T) {
	t.Parallel()

	var (
		signer1 = common.HexToAddress("0x1")
		signer2 = common.HexToAddress("0x2")
	)

	tests := []struct {
		name    string
		give    bindings.MultisigConfig
		want    *types.Config
		wantErr string
	}{
		{
			name: "success: converts binding config to config",
			give: bindings.MultisigConfig{
				GroupQuorums: [32]uint8{1, 1},
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.McmSigner{
					{EvmAddress: signer1, Group: 0, Index: 0},
					{EvmAddress: signer2, Group: 1, Index: 1},
				},
			},
			want: &types.Config{
				Quorum:  1,
				Signers: []common.Address{signer1},
				GroupSigners: []types.Config{
					{
						Quorum:       1,
						Signers:      []common.Address{signer2},
						GroupSigners: []types.Config{},
					},
				},
			},
		},
		{
			name: "failure: validation error on resulting config",
			give: bindings.MultisigConfig{
				GroupQuorums: [32]uint8{0, 1}, // A zero quorum makes this invalid
				GroupParents: [32]uint8{0, 0},
				Signers: []bindings.McmSigner{
					{EvmAddress: signer1, Group: 0, Index: 0},
					{EvmAddress: signer2, Group: 1, Index: 1},
				},
			},
			wantErr: "invalid MCMS config: Quorum must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transformer := ConfigTransformer{}
			got, err := transformer.ToConfig(&tt.give)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
