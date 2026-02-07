//go:build e2e

package canton

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/noders-team/go-daml/pkg/types"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

func TestToConfig(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       mcms.MultisigConfig
		expected    mcmstypes.Config
	}{
		{
			name:        "simple_2of3",
			description: "Simple 2-of-3 multisig with all signers in root group (group 0)",
			input: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{SignerAddress: types.TEXT("0x1111111111111111111111111111111111111111"), SignerIndex: types.INT64(0), SignerGroup: types.INT64(0)},
					{SignerAddress: types.TEXT("0x2222222222222222222222222222222222222222"), SignerIndex: types.INT64(1), SignerGroup: types.INT64(0)},
					{SignerAddress: types.TEXT("0x3333333333333333333333333333333333333333"), SignerIndex: types.INT64(2), SignerGroup: types.INT64(0)},
				},
				GroupQuorums: []types.INT64{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				GroupParents: []types.INT64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			expected: mcmstypes.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("1111111111111111111111111111111111111111"),
					common.HexToAddress("2222222222222222222222222222222222222222"),
					common.HexToAddress("3333333333333333333333333333333333333333"),
				},
				GroupSigners: []mcmstypes.Config{},
			},
		},
		{
			name:        "hierarchical_2level",
			description: "2-level hierarchy: root group 0 has 1 direct signer + group 1 as child. Group 1 has 3 signers with quorum 2. Root quorum is 1 (can be satisfied by direct signer OR group 1 reaching quorum).",
			input: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{SignerAddress: types.TEXT("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), SignerIndex: types.INT64(0), SignerGroup: types.INT64(0)},
					{SignerAddress: types.TEXT("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"), SignerIndex: types.INT64(1), SignerGroup: types.INT64(1)},
					{SignerAddress: types.TEXT("0xcccccccccccccccccccccccccccccccccccccccc"), SignerIndex: types.INT64(2), SignerGroup: types.INT64(1)},
					{SignerAddress: types.TEXT("0xdddddddddddddddddddddddddddddddddddddddd"), SignerIndex: types.INT64(3), SignerGroup: types.INT64(1)},
				},
				GroupQuorums: []types.INT64{1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				GroupParents: []types.INT64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			expected: mcmstypes.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				},
				GroupSigners: []mcmstypes.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
							common.HexToAddress("cccccccccccccccccccccccccccccccccccccccc"),
							common.HexToAddress("dddddddddddddddddddddddddddddddddddddddd"),
						},
						GroupSigners: []mcmstypes.Config{},
					},
				},
			},
		},
		{
			name:        "complex_3level",
			description: "3-level hierarchy: Group 0 (root) quorum 2, Group 1 (parent 0) quorum 2, Group 2 (parent 0) quorum 1, Group 3 (parent 1) quorum 2. Tests deeper nesting with multiple child groups at same level.",
			input: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000001"), SignerIndex: types.INT64(0), SignerGroup: types.INT64(0)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000002"), SignerIndex: types.INT64(1), SignerGroup: types.INT64(1)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000003"), SignerIndex: types.INT64(2), SignerGroup: types.INT64(1)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000004"), SignerIndex: types.INT64(3), SignerGroup: types.INT64(2)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000005"), SignerIndex: types.INT64(4), SignerGroup: types.INT64(2)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000006"), SignerIndex: types.INT64(5), SignerGroup: types.INT64(3)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000007"), SignerIndex: types.INT64(6), SignerGroup: types.INT64(3)},
					{SignerAddress: types.TEXT("0x1000000000000000000000000000000000000008"), SignerIndex: types.INT64(7), SignerGroup: types.INT64(3)},
				},
				GroupQuorums: []types.INT64{2, 2, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				GroupParents: []types.INT64{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			expected: mcmstypes.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("1000000000000000000000000000000000000001"),
				},
				GroupSigners: []mcmstypes.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("1000000000000000000000000000000000000002"),
							common.HexToAddress("1000000000000000000000000000000000000003"),
						},
						GroupSigners: []mcmstypes.Config{
							{
								Quorum: 2,
								Signers: []common.Address{
									common.HexToAddress("1000000000000000000000000000000000000006"),
									common.HexToAddress("1000000000000000000000000000000000000007"),
									common.HexToAddress("1000000000000000000000000000000000000008"),
								},
								GroupSigners: []mcmstypes.Config{},
							},
						},
					},
					{
						Quorum: 1,
						Signers: []common.Address{
							common.HexToAddress("1000000000000000000000000000000000000004"),
							common.HexToAddress("1000000000000000000000000000000000000005"),
						},
						GroupSigners: []mcmstypes.Config{},
					},
				},
			},
		},
		{
			name:        "empty_groups_edge_case",
			description: "Edge case: groups with quorum 0 (disabled) interspersed with active groups. Group 0 active (quorum 1), Group 1 disabled (quorum 0), Group 2 active (quorum 2, parent 0). The toConfig function should skip disabled groups.",
			input: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{SignerAddress: types.TEXT("0xdead000000000000000000000000000000000001"), SignerIndex: types.INT64(0), SignerGroup: types.INT64(0)},
					{SignerAddress: types.TEXT("0xdead000000000000000000000000000000000002"), SignerIndex: types.INT64(1), SignerGroup: types.INT64(2)},
					{SignerAddress: types.TEXT("0xdead000000000000000000000000000000000003"), SignerIndex: types.INT64(2), SignerGroup: types.INT64(2)},
				},
				GroupQuorums: []types.INT64{1, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				GroupParents: []types.INT64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			expected: mcmstypes.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("dead000000000000000000000000000000000001"),
				},
				GroupSigners: []mcmstypes.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("dead000000000000000000000000000000000002"),
							common.HexToAddress("dead000000000000000000000000000000000003"),
						},
						GroupSigners: []mcmstypes.Config{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toConfig(tt.input)
			require.NoError(t, err, tt.description)
			require.NotNil(t, result)

			// Compare the result with expected
			require.Equal(t, tt.expected.Quorum, result.Quorum, "quorum mismatch")
			require.Equal(t, len(tt.expected.Signers), len(result.Signers), "signers count mismatch")

			// Compare signers
			for i, expectedSigner := range tt.expected.Signers {
				require.Equal(t, expectedSigner, result.Signers[i], "signer mismatch at index %d", i)
			}

			// Compare group signers recursively
			compareGroupSigners(t, tt.expected.GroupSigners, result.GroupSigners)
		})
	}
}

func compareGroupSigners(t *testing.T, expected, actual []mcmstypes.Config) {
	require.Equal(t, len(expected), len(actual), "group signers count mismatch")

	for i := range expected {
		require.Equal(t, expected[i].Quorum, actual[i].Quorum, "group %d quorum mismatch", i)
		require.Equal(t, len(expected[i].Signers), len(actual[i].Signers), "group %d signers count mismatch", i)

		for j, expectedSigner := range expected[i].Signers {
			require.Equal(t, expectedSigner, actual[i].Signers[j], "group %d signer mismatch at index %d", i, j)
		}

		// Recursively compare nested group signers
		compareGroupSigners(t, expected[i].GroupSigners, actual[i].GroupSigners)
	}
}
