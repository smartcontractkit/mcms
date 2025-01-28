package evm

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
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
		give    bindings.ManyChainMultiSigConfig
		want    *types.Config
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
			name: "success: nested configs",
			give: bindings.ManyChainMultiSigConfig{
				GroupQuorums: [32]uint8{2, 4, 1, 1, 3, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				GroupParents: [32]uint8{0, 0, 1, 2, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Signers: []bindings.ManyChainMultiSigSigner{
					{Addr: common.HexToAddress("0x156a4DB782D04911FFd880885e04Fdb3611084cf"), Index: 0, Group: 0},
					{Addr: common.HexToAddress("0x16e1Ff31F5890C75E737703883b6696966d40bef"), Index: 1, Group: 0},
					{Addr: common.HexToAddress("0x34eDf371A1489Fc463579De16af39502eeF0D58C"), Index: 2, Group: 0},
					{Addr: common.HexToAddress("0x3A7942bB5A8597769a7D32698fe58483f6D983c9"), Index: 3, Group: 1},
					{Addr: common.HexToAddress("0x3e87E9D6fbe4660e55150228E600b2f3bfd25463"), Index: 4, Group: 1},
					{Addr: common.HexToAddress("0x5262971Ba093A8ce9D739A3Cbe860319EA181eAF"), Index: 5, Group: 1},
					{Addr: common.HexToAddress("0x63c49Cc2075cE6050E2413De66181dE456A99982"), Index: 6, Group: 1},
					{Addr: common.HexToAddress("0x666bC8eD0Ac152f60e2A8EBEE68528940af3370F"), Index: 7, Group: 1},
					{Addr: common.HexToAddress("0x67530Eb5B40a4279a21AAcace0192417c1040956"), Index: 8, Group: 2},
					{Addr: common.HexToAddress("0x9ACc926BbD7fd76Ac9f19295D4cb0769e3f0bD43"), Index: 9, Group: 2},
					{Addr: common.HexToAddress("0xAC3cd4fE42fEd3CeBad0D8B1164D048FeF46376f"), Index: 10, Group: 3},
					{Addr: common.HexToAddress("0xc6E315d96A4dD15F3B844fEA41cc00E37aF0830d"), Index: 11, Group: 4},
					{Addr: common.HexToAddress("0xd30dDc8cD88e4B4843BAC08B9551c2c91DCD44d6"), Index: 12, Group: 4},
					{Addr: common.HexToAddress("0xE0b8e2CCe74082197423C4d6f4232c4316133A35"), Index: 13, Group: 4},
					{Addr: common.HexToAddress("0xEcB7cd3f3FDb405e1Dfa24Ba1565F19fe65Bc460"), Index: 14, Group: 4},
					{Addr: common.HexToAddress("0xFCc93a2f3d4511Db830b079A3c8B7c96594b232B"), Index: 15, Group: 5},
				},
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("0x156a4DB782D04911FFd880885e04Fdb3611084cf"),
					common.HexToAddress("0x16e1Ff31F5890C75E737703883b6696966d40bef"),
					common.HexToAddress("0x34eDf371A1489Fc463579De16af39502eeF0D58C"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 4,
						Signers: []common.Address{
							common.HexToAddress("0x3A7942bB5A8597769a7D32698fe58483f6D983c9"),
							common.HexToAddress("0x3e87E9D6fbe4660e55150228E600b2f3bfd25463"),
							common.HexToAddress("0x5262971Ba093A8ce9D739A3Cbe860319EA181eAF"),
							common.HexToAddress("0x63c49Cc2075cE6050E2413De66181dE456A99982"),
							common.HexToAddress("0x666bC8eD0Ac152f60e2A8EBEE68528940af3370F"),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									common.HexToAddress("0x67530Eb5B40a4279a21AAcace0192417c1040956"),
									common.HexToAddress("0x9ACc926BbD7fd76Ac9f19295D4cb0769e3f0bD43"),
								},
								GroupSigners: []types.Config{
									{
										Quorum: 1,
										Signers: []common.Address{
											common.HexToAddress("0xAC3cd4fE42fEd3CeBad0D8B1164D048FeF46376f"),
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
							common.HexToAddress("0xc6E315d96A4dD15F3B844fEA41cc00E37aF0830d"),
							common.HexToAddress("0xd30dDc8cD88e4B4843BAC08B9551c2c91DCD44d6"),
							common.HexToAddress("0xE0b8e2CCe74082197423C4d6f4232c4316133A35"),
							common.HexToAddress("0xEcB7cd3f3FDb405e1Dfa24Ba1565F19fe65Bc460"),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 1,
								Signers: []common.Address{
									common.HexToAddress("0xFCc93a2f3d4511Db830b079A3c8B7c96594b232B"),
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

			transformer := ConfigTransformer{}
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

	var (
		signer1 = common.HexToAddress("0x1")
		signer2 = common.HexToAddress("0x2")
		signer3 = common.HexToAddress("0x3")
		signer4 = common.HexToAddress("0x4")
		signer5 = common.HexToAddress("0x5")
	)

	tests := []struct {
		name       string
		giveConfig types.Config
		want       bindings.ManyChainMultiSigConfig
		wantErr    string
	}{
		{
			name: "success: root signers with some groups",
			giveConfig: types.Config{
				Quorum:  1,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []types.Config{
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
			giveConfig: types.Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []types.Config{
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
			giveConfig: types.Config{
				Quorum:       1,
				Signers:      []common.Address{signer1, signer2},
				GroupSigners: []types.Config{},
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
			giveConfig: types.Config{
				Quorum:  2,
				Signers: []common.Address{},
				GroupSigners: []types.Config{
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
			giveConfig: types.Config{
				Quorum:  2,
				Signers: []common.Address{signer1, signer2},
				GroupSigners: []types.Config{
					{
						Quorum:  1,
						Signers: []common.Address{signer3},
						GroupSigners: []types.Config{
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
			giveConfig: types.Config{
				Quorum: 2,
				// Root signers are out of order (signer2 is before signer1)
				Signers: []common.Address{signer2, signer1},
				// Group signers are out of order (signer5 is before the signer4 group)
				GroupSigners: []types.Config{
					{
						Quorum:  1,
						Signers: []common.Address{signer3},
						GroupSigners: []types.Config{
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

			transformer := ConfigTransformer{}
			got, err := transformer.ToChainConfig(tt.giveConfig)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
