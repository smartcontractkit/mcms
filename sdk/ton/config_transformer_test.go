package ton_test

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
)

func makeRandomTestWallet(api wallet.TonAPI, networkGlobalID int32) (*wallet.Wallet, error) {
	v5r1Config := wallet.ConfigV5R1Final{
		NetworkGlobalID: networkGlobalID,
		Workchain:       0,
	}
	return wallet.FromSeed(api, wallet.NewSeed(), v5r1Config)
}

func PublicKeyToBigInt(pub crypto.PublicKey) (*big.Int, error) {
	pubEd, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ed25519 key")
	}

	// TODO: currently only works with 20 byte keys bc types.Config.Signers is [20]byte
	return new(big.Int).SetBytes(pubEd[:20]), nil
}

func mustKey(w *wallet.Wallet) *big.Int {
	return must(PublicKeyToBigInt(w.PrivateKey().Public()))
}

func Test_ConfigTransformer_ToConfig(t *testing.T) {
	t.Parallel()

	chainID := chaintest.Chain7TONID
	var client *ton.APIClient = nil
	wallets := []*wallet.Wallet{
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
	}

	// TODO: we should also sort keys asc on this side

	var (
		signer1 = wallets[0]
		signer2 = wallets[1]
	)

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
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 1, Index: 1},
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
				Signers: []common.Address{common.Address(mustKey(signer1).Bytes())},
				GroupSigners: []types.Config{
					{
						Quorum:       1,
						Signers:      []common.Address{common.Address(mustKey(signer2).Bytes())},
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
					{Key: mustKey(wallets[0]), Index: 0, Group: 0},
					{Key: mustKey(wallets[1]), Index: 1, Group: 0},
					{Key: mustKey(wallets[2]), Index: 2, Group: 0},
					{Key: mustKey(wallets[3]), Index: 3, Group: 1},
					{Key: mustKey(wallets[4]), Index: 4, Group: 1},
					{Key: mustKey(wallets[5]), Index: 5, Group: 1},
					{Key: mustKey(wallets[6]), Index: 6, Group: 1},
					{Key: mustKey(wallets[7]), Index: 7, Group: 1},
					{Key: mustKey(wallets[8]), Index: 8, Group: 2},
					{Key: mustKey(wallets[9]), Index: 9, Group: 2},
					{Key: mustKey(wallets[10]), Index: 10, Group: 3},
					{Key: mustKey(wallets[11]), Index: 11, Group: 4},
					{Key: mustKey(wallets[12]), Index: 12, Group: 4},
					{Key: mustKey(wallets[13]), Index: 13, Group: 4},
					{Key: mustKey(wallets[14]), Index: 14, Group: 4},
					{Key: mustKey(wallets[15]), Index: 15, Group: 5},
				}, tvm.KeyUINT8)),
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.Address(mustKey(wallets[0]).Bytes()),
					common.Address(mustKey(wallets[1]).Bytes()),
					common.Address(mustKey(wallets[2]).Bytes()),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 4,
						Signers: []common.Address{
							common.Address(mustKey(wallets[3]).Bytes()),
							common.Address(mustKey(wallets[4]).Bytes()),
							common.Address(mustKey(wallets[5]).Bytes()),
							common.Address(mustKey(wallets[6]).Bytes()),
							common.Address(mustKey(wallets[7]).Bytes()),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									common.Address(mustKey(wallets[8]).Bytes()),
									common.Address(mustKey(wallets[9]).Bytes()),
								},
								GroupSigners: []types.Config{
									{
										Quorum: 1,
										Signers: []common.Address{
											common.Address(mustKey(wallets[10]).Bytes()),
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
							common.Address(mustKey(wallets[11]).Bytes()),
							common.Address(mustKey(wallets[12]).Bytes()),
							common.Address(mustKey(wallets[13]).Bytes()),
							common.Address(mustKey(wallets[14]).Bytes()),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									common.Address(mustKey(wallets[15]).Bytes()),
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
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 1, Index: 1},
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

	chainID := chaintest.Chain7TONID
	var client *ton.APIClient = nil
	wallets := []*wallet.Wallet{
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
		must(makeRandomTestWallet(client, chainID)),
	}

	// Sort signers by their pub keys in ascending order
	slices.SortFunc(wallets, func(i, j *wallet.Wallet) int {
		return mustKey(i).Cmp(mustKey(j))
	})

	var (
		signer1 = wallets[0]
		signer2 = wallets[1]
		signer3 = wallets[2]
		signer4 = wallets[3]
		signer5 = wallets[4]
	)

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
					common.Address(mustKey(signer1).Bytes()),
					common.Address(mustKey(signer2).Bytes()),
				},
				GroupSigners: []types.Config{
					{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer3).Bytes())}},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 0, Index: 1},
					{Key: mustKey(signer3), Group: 1, Index: 2},
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
					common.Address(mustKey(signer1).Bytes()),
					common.Address(mustKey(signer2).Bytes()),
				},
				GroupSigners: []types.Config{
					{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer3).Bytes())}},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 0, Index: 1},
					{Key: mustKey(signer3), Group: 1, Index: 2},
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
					common.Address(mustKey(signer1).Bytes()),
					common.Address(mustKey(signer2).Bytes()),
				},
				GroupSigners: []types.Config{},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 0, Index: 1},
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
					{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer1).Bytes())}},
					{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer2).Bytes())}},
					{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer3).Bytes())}},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(signer1), Group: 1, Index: 0},
					{Key: mustKey(signer2), Group: 2, Index: 1},
					{Key: mustKey(signer3), Group: 3, Index: 2},
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
					common.Address(mustKey(signer1).Bytes()),
					common.Address(mustKey(signer2).Bytes()),
				},
				GroupSigners: []types.Config{
					{
						Quorum:  1,
						Signers: []common.Address{common.Address(mustKey(signer3).Bytes())},
						GroupSigners: []types.Config{
							{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer4).Bytes())}},
						},
					},
					{
						Quorum:  1,
						Signers: []common.Address{common.Address(mustKey(signer5).Bytes())},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 0, Index: 1},
					{Key: mustKey(signer3), Group: 1, Index: 2},
					{Key: mustKey(signer4), Group: 2, Index: 3},
					{Key: mustKey(signer5), Group: 3, Index: 4},
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
					common.Address(mustKey(signer2).Bytes()),
					common.Address(mustKey(signer1).Bytes()),
				},
				// Group signers are out of order (signer5 is before the signer4 group)
				GroupSigners: []types.Config{
					{
						Quorum:  1,
						Signers: []common.Address{common.Address(mustKey(signer3).Bytes())},
						GroupSigners: []types.Config{
							{Quorum: 1, Signers: []common.Address{common.Address(mustKey(signer5).Bytes())}},
						},
					},
					{
						Quorum:  1,
						Signers: []common.Address{common.Address(mustKey(signer4).Bytes())},
					},
				},
			},
			want: mcms.Config{
				Signers: must(tvm.MakeDictFrom([]mcms.Signer{
					{Key: mustKey(signer1), Group: 0, Index: 0},
					{Key: mustKey(signer2), Group: 0, Index: 1},
					{Key: mustKey(signer3), Group: 1, Index: 2},
					{Key: mustKey(signer4), Group: 3, Index: 3},
					{Key: mustKey(signer5), Group: 2, Index: 4},
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
