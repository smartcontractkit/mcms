package ton_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
)

func TestConfigTransformer_ToConfig(t *testing.T) {
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
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 1, Index: 1},
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
					{Address: AsUint160Addr(signers[0]), Index: 0, Group: 0},
					{Address: AsUint160Addr(signers[1]), Index: 1, Group: 0},
					{Address: AsUint160Addr(signers[2]), Index: 2, Group: 0},
					{Address: AsUint160Addr(signers[3]), Index: 3, Group: 1},
					{Address: AsUint160Addr(signers[4]), Index: 4, Group: 1},
					{Address: AsUint160Addr(signers[5]), Index: 5, Group: 1},
					{Address: AsUint160Addr(signers[6]), Index: 6, Group: 1},
					{Address: AsUint160Addr(signers[7]), Index: 7, Group: 1},
					{Address: AsUint160Addr(signers[8]), Index: 8, Group: 2},
					{Address: AsUint160Addr(signers[9]), Index: 9, Group: 2},
					{Address: AsUint160Addr(signers[10]), Index: 10, Group: 3},
					{Address: AsUint160Addr(signers[11]), Index: 11, Group: 4},
					{Address: AsUint160Addr(signers[12]), Index: 12, Group: 4},
					{Address: AsUint160Addr(signers[13]), Index: 13, Group: 4},
					{Address: AsUint160Addr(signers[14]), Index: 14, Group: 4},
					{Address: AsUint160Addr(signers[15]), Index: 15, Group: 5},
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
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 1, Index: 1},
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

			transformer := mcmston.NewConfigTransformer()
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

func TestSetConfigInputs(t *testing.T) {
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
					{
						Quorum:       1,
						Signers:      []common.Address{signers[2].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 0, Index: 1},
					{Address: AsUint160Addr(signers[2]), Group: 1, Index: 2},
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
					{
						Quorum:       1,
						Signers:      []common.Address{signers[2].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 0, Index: 1},
					{Address: AsUint160Addr(signers[2]), Group: 1, Index: 2},
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
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 0, Index: 1},
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
					{
						Quorum:       1,
						Signers:      []common.Address{signers[0].Address()},
						GroupSigners: []types.Config{},
					},
					{
						Quorum:       1,
						Signers:      []common.Address{signers[1].Address()},
						GroupSigners: []types.Config{},
					},
					{
						Quorum:       1,
						Signers:      []common.Address{signers[2].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: AsUint160Addr(signers[0]), Group: 1, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 2, Index: 1},
					{Address: AsUint160Addr(signers[2]), Group: 3, Index: 2},
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
							{
								Quorum:       1,
								Signers:      []common.Address{signers[3].Address()},
								GroupSigners: []types.Config{},
							},
						},
					},
					{
						Quorum:       1,
						Signers:      []common.Address{signers[4].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 0, Index: 1},
					{Address: AsUint160Addr(signers[2]), Group: 1, Index: 2},
					{Address: AsUint160Addr(signers[3]), Group: 2, Index: 3},
					{Address: AsUint160Addr(signers[4]), Group: 3, Index: 4},
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
							{
								Quorum:       1,
								Signers:      []common.Address{signers[4].Address()},
								GroupSigners: []types.Config{},
							},
						},
					},
					{
						Quorum:       1,
						Signers:      []common.Address{signers[3].Address()},
						GroupSigners: []types.Config{},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Address: AsUint160Addr(signers[0]), Group: 0, Index: 0},
					{Address: AsUint160Addr(signers[1]), Group: 0, Index: 1},
					{Address: AsUint160Addr(signers[2]), Group: 1, Index: 2},
					{Address: AsUint160Addr(signers[3]), Group: 3, Index: 3},
					{Address: AsUint160Addr(signers[4]), Group: 2, Index: 4},
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

			transformer := mcmston.NewConfigTransformer()
			got, err := transformer.ToChainConfig(tt.giveConfig, nil)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)

				recovered, err := transformer.ToConfig(got)
				require.NoError(t, err)
				// Notice: compare .GroupSigners to avoid issues with recovering case unsorted_signers_and_groups
				assert.Equal(t, tt.giveConfig.GroupSigners, recovered.GroupSigners)
			}
		})
	}
}
