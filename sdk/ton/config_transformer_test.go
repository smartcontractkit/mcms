package ton_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/types"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

func Test_ConfigTransformer_ToConfig(t *testing.T) {
	t.Parallel()

	signers := testutils.MakeNewECDSASigners(16)

	tests := []struct {
		name    string
		give    mcms.Config
		want    *types.Config
		wantErr string
	}{
		{
			name: "success: converts binding config to config",
			give: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 1, Index: 1},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 1},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
			want: &types.Config{
				Quorum:  1,
				Signers: []common.Address{signers[0].Address()},
				GroupSigners: []types.Config{
					{
						Quorum:       1,
						Signers:      []common.Address{signers[1].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
		},
		{
			name: "success: nested configs",
			give: mcms.Config{
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 2},
					{Val: 4},
					{Val: 1},
					{Val: 1},
					{Val: 3},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
					{Val: 1},
					{Val: 2},
					{Val: 0},
					{Val: 4},
				}, tvm.KeyUINT8)),
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Index: 0, Group: 0},
					{Address: signers[1].Address().Big(), Index: 1, Group: 0},
					{Address: signers[2].Address().Big(), Index: 2, Group: 0},
					{Address: signers[3].Address().Big(), Index: 3, Group: 1},
					{Address: signers[4].Address().Big(), Index: 4, Group: 1},
					{Address: signers[5].Address().Big(), Index: 5, Group: 1},
					{Address: signers[6].Address().Big(), Index: 6, Group: 1},
					{Address: signers[7].Address().Big(), Index: 7, Group: 1},
					{Address: signers[8].Address().Big(), Index: 8, Group: 2},
					{Address: signers[9].Address().Big(), Index: 9, Group: 2},
					{Address: signers[10].Address().Big(), Index: 10, Group: 3},
					{Address: signers[11].Address().Big(), Index: 11, Group: 4},
					{Address: signers[12].Address().Big(), Index: 12, Group: 4},
					{Address: signers[13].Address().Big(), Index: 13, Group: 4},
					{Address: signers[14].Address().Big(), Index: 14, Group: 4},
					{Address: signers[15].Address().Big(), Index: 15, Group: 5},
				}, tvm.KeyUINT8)),
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[0].Address(),
					signers[1].Address(),
					signers[2].Address(),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 4,
						Signers: []common.Address{
							signers[3].Address(),
							signers[4].Address(),
							signers[5].Address(),
							signers[6].Address(),
							signers[7].Address(),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									signers[8].Address(),
									signers[9].Address(),
								},
								GroupSigners: []types.Config{
									{
										Quorum: 1,
										Signers: []common.Address{
											signers[10].Address(),
										},
										GroupSigners: []types.Config{},
									},
								},
							},
						},
					},
					{
						Quorum: 3,
						Signers: []common.Address{
							signers[11].Address(),
							signers[12].Address(),
							signers[13].Address(),
							signers[14].Address(),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									signers[15].Address(),
								},
								GroupSigners: []types.Config{},
							},
						},
					},
				},
			},
		},
		{
			name: "failure: validation error on resulting config",
			give: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 1, Index: 1},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 0}, // A zero quorum makes this invalid
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
			wantErr: "invalid MCMS config: Quorum must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transformer := tonmcms.NewConfigTransformer()
			got, err := transformer.ToConfig(tt.give)

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

	signers := testutils.MakeNewECDSASigners(5)

	tests := []struct {
		name       string
		giveConfig types.Config
		want       mcms.Config
		wantErr    string
	}{
		{
			name: "success: root signers with some groups",
			giveConfig: types.Config{
				Quorum: 1,
				Signers: []common.Address{
					signers[0].Address(),
					signers[1].Address(),
				},
				GroupSigners: []types.Config{
					{Quorum: 1, Signers: []common.Address{signers[2].Address()}},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 0, Index: 1},
					{Address: signers[2].Address().Big(), Group: 1, Index: 2},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 1},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
		},
		{
			name: "success: root signers with some groups and increased quorum",
			giveConfig: types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[0].Address(),
					signers[1].Address(),
				},
				GroupSigners: []types.Config{
					{Quorum: 1, Signers: []common.Address{signers[2].Address()}},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 0, Index: 1},
					{Address: signers[2].Address().Big(), Group: 1, Index: 2},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 2},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
		},
		{
			name: "success: only root signers",
			giveConfig: types.Config{
				Quorum: 1,
				Signers: []common.Address{
					signers[0].Address(),
					signers[1].Address(),
				},
				GroupSigners: []types.Config{},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 0, Index: 1},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
		},
		{
			name: "success: only groups",
			giveConfig: types.Config{
				Quorum:  2,
				Signers: []common.Address{},
				GroupSigners: []types.Config{
					{Quorum: 1, Signers: []common.Address{signers[0].Address()}},
					{Quorum: 1, Signers: []common.Address{signers[1].Address()}},
					{Quorum: 1, Signers: []common.Address{signers[2].Address()}},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 1, Index: 0},
					{Address: signers[1].Address().Big(), Group: 2, Index: 1},
					{Address: signers[2].Address().Big(), Group: 3, Index: 2},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 2},
					{Val: 1},
					{Val: 1},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
					{Val: 0},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
		},
		{
			name: "success: nested signers and groups",
			giveConfig: types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[0].Address(),
					signers[1].Address(),
				},
				GroupSigners: []types.Config{
					{
						Quorum:  1,
						Signers: []common.Address{signers[2].Address()},
						GroupSigners: []types.Config{
							{Quorum: 1, Signers: []common.Address{signers[3].Address()}},
						},
					},
					{
						Quorum:  1,
						Signers: []common.Address{signers[4].Address()},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 0, Index: 1},
					{Address: signers[2].Address().Big(), Group: 1, Index: 2},
					{Address: signers[3].Address().Big(), Group: 2, Index: 3},
					{Address: signers[4].Address().Big(), Group: 3, Index: 4},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 2},
					{Val: 1},
					{Val: 1},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
					{Val: 1},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
		},
		{
			name: "success: unsorted signers and groups",
			giveConfig: types.Config{
				Quorum: 2,
				// Root signers are out of order (signer2 is before signer1)
				Signers: []common.Address{
					signers[1].Address(),
					signers[0].Address(),
				},
				// Group signers are out of order (signer5 is before the signer4 group)
				GroupSigners: []types.Config{
					{
						Quorum:  1,
						Signers: []common.Address{signers[2].Address()},
						GroupSigners: []types.Config{
							{Quorum: 1, Signers: []common.Address{signers[4].Address()}},
						},
					},
					{
						Quorum:  1,
						Signers: []common.Address{signers[3].Address()},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: signers[0].Address().Big(), Group: 0, Index: 0},
					{Address: signers[1].Address().Big(), Group: 0, Index: 1},
					{Address: signers[2].Address().Big(), Group: 1, Index: 2},
					{Address: signers[3].Address().Big(), Group: 3, Index: 3},
					{Address: signers[4].Address().Big(), Group: 2, Index: 4},
				}, tvm.KeyUINT8)),
				GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{
					{Val: 2},
					{Val: 1},
					{Val: 1},
					{Val: 1},
				}, tvm.KeyUINT8)),
				GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{
					{Val: 0},
					{Val: 0},
					{Val: 1},
					{Val: 0},
				}, tvm.KeyUINT8)),
			},
		},
		{
			name: "failure: signer count cannot exceed 255",
			giveConfig: types.Config{
				Quorum:  1,
				Signers: make([]common.Address, math.MaxUint8+1),
			},
			wantErr: "too many signers: 256 max number is 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transformer := tonmcms.NewConfigTransformer()
			got, err := transformer.ToChainConfig(tt.giveConfig, nil)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
