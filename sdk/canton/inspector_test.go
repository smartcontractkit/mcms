package canton

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	mcmsapi "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms/api"
	mcmscore "github.com/smartcontractkit/chainlink-canton/bindings/generated/latest/mcms/core"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	damltypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/stretchr/testify/require"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

const testParty = "participant1-localparty-1::1220acd0401d95915bef2f498a45cc8f3c43119dde50cf370864e9aa4eb03d817cfb"

func testMCMSContract(instanceID string, roleState func(mcmsapi.RoleState) mcmsapi.RoleState) mcmscore.MCMS {
	proposer := mcmsapi.RoleState{
		Config: mcmsapi.MultisigConfig{
			Signers:      []mcmsapi.SignerInfo{},
			GroupQuorums: []damltypes.INT64{damltypes.INT64(1)},
			GroupParents: []damltypes.INT64{damltypes.INT64(0)},
		},
		ExpiringRoot: mcmsapi.ExpiringRoot{
			Root:      damltypes.TEXT("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			OpCount:   damltypes.INT64(7),
			ValidUntil: damltypes.TIMESTAMP(time.Unix(1_700_000_000, 0)),
		},
		RootMetadata: mcmsapi.RootMetadata{
			ChainId:    damltypes.INT64(1),
			MultisigId: damltypes.TEXT(instanceID + "@" + testParty + "-proposer"),
		},
	}
	if roleState != nil {
		proposer = roleState(proposer)
	}

	return mcmscore.MCMS{
		Owner:      damltypes.PARTY(testParty),
		InstanceId: damltypes.TEXT(instanceID),
		ChainId:    damltypes.INT64(1),
		Proposer:   proposer,
		Bypasser: mcmsapi.RoleState{
			ExpiringRoot: mcmsapi.ExpiringRoot{OpCount: damltypes.INT64(3)},
			RootMetadata: mcmsapi.RootMetadata{
				ChainId:    damltypes.INT64(1),
				MultisigId: damltypes.TEXT(instanceID + "@" + testParty + "-bypasser"),
			},
		},
		Canceller: mcmsapi.RoleState{
			ExpiringRoot: mcmsapi.ExpiringRoot{OpCount: damltypes.INT64(5)},
			RootMetadata: mcmsapi.RootMetadata{
				ChainId:    damltypes.INT64(1),
				MultisigId: damltypes.TEXT(instanceID + "@" + testParty + "-canceller"),
			},
		},
	}
}

func TestExpiringRootOpCount(t *testing.T) {
	t.Parallel()

	contract := testMCMSContract(mcmsInstanceIDCCIP, nil)

	opCount, err := expiringRootOpCount(&contract, TimelockRoleProposer)
	require.NoError(t, err)
	require.Equal(t, uint64(7), opCount)

	opCount, err = expiringRootOpCount(&contract, TimelockRoleBypasser)
	require.NoError(t, err)
	require.Equal(t, uint64(3), opCount)

	_, err = expiringRootOpCount(&contract, TimelockRole(99))
	require.ErrorContains(t, err, "unknown timelock role")
}

func TestRootFromExpiringRoot(t *testing.T) {
	t.Parallel()

	wantRoot := common.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	expiringRoot := mcmsapi.ExpiringRoot{
		Root:       damltypes.TEXT(wantRoot.Hex()),
		ValidUntil: damltypes.TIMESTAMP(time.Unix(1_700_000_000, 0)),
	}

	root, validUntil, err := rootFromExpiringRoot(expiringRoot)
	require.NoError(t, err)
	require.Equal(t, wantRoot, root)
	require.Equal(t, uint32(1_700_000_000), validUntil)

	_, _, err = rootFromExpiringRoot(mcmsapi.ExpiringRoot{
		ValidUntil: damltypes.TIMESTAMP(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC)),
	})
	require.ErrorContains(t, err, "valid until out of range")
}

func TestExpiringRootForRole(t *testing.T) {
	t.Parallel()

	contract := testMCMSContract(mcmsInstanceIDCCIP, nil)

	root, err := expiringRootForRole(&contract, TimelockRoleProposer)
	require.NoError(t, err)
	require.Equal(t, damltypes.INT64(7), root.OpCount)

	_, err = expiringRootForRole(&contract, TimelockRole(42))
	require.ErrorContains(t, err, "unknown timelock role")
}

func TestChainMetadataFromMCMSContract(t *testing.T) {
	t.Parallel()

	contract := testMCMSContract(mcmsInstanceIDCCIP, nil)
	wantAddress := contracts.InstanceID(mcmsInstanceIDCCIP).RawInstanceAddress(damltypes.PARTY(testParty)).InstanceAddress().Hex()

	meta, err := chainMetadataFromMCMSContract(&contract, TimelockRoleProposer)
	require.NoError(t, err)
	require.Equal(t, uint64(7), meta.StartingOpCount)
	require.Equal(t, wantAddress, meta.MCMAddress)

	var fields AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(meta.AdditionalFields, &fields))
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, mcmsInstanceIDCCIP+"@"+testParty+"-proposer", fields.MultisigId)
	require.Equal(t, mcmsInstanceIDCCIP, fields.InstanceId)

	meta, err = chainMetadataFromMCMSContract(&contract, TimelockRoleBypasser)
	require.NoError(t, err)
	require.Equal(t, uint64(3), meta.StartingOpCount)

	freshRoot := testMCMSContract(mcmsInstanceIDCCIP, func(rs mcmsapi.RoleState) mcmsapi.RoleState {
		rs.RootMetadata = mcmsapi.RootMetadata{}
		return rs
	})
	meta, err = chainMetadataFromMCMSContract(&freshRoot, TimelockRoleProposer)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(meta.AdditionalFields, &fields))
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, mcmsInstanceIDCCIP+"@"+testParty+"-proposer", fields.MultisigId)

	noChainID := freshRoot
	noChainID.ChainId = damltypes.INT64(0)
	_, err = chainMetadataFromMCMSContract(&noChainID, TimelockRoleProposer)
	require.ErrorContains(t, err, "invalid root metadata from ledger")
}

func TestToConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		description string
		input       mcmsapi.MultisigConfig
		expected    mcmstypes.Config
	}{
		{
			name:        "simple_2of3",
			description: "Simple 2-of-3 multisig with all signers in root group (group 0)",
			input: mcmsapi.MultisigConfig{
				Signers: []mcmsapi.SignerInfo{
					{SignerAddress: damltypes.TEXT("0x1111111111111111111111111111111111111111"), SignerIndex: damltypes.INT64(0), SignerGroup: damltypes.INT64(0)},
					{SignerAddress: damltypes.TEXT("0x2222222222222222222222222222222222222222"), SignerIndex: damltypes.INT64(1), SignerGroup: damltypes.INT64(0)},
					{SignerAddress: damltypes.TEXT("0x3333333333333333333333333333333333333333"), SignerIndex: damltypes.INT64(2), SignerGroup: damltypes.INT64(0)},
				},
				GroupQuorums: repeatInt64(2, 0),
				GroupParents: repeatInt64(0, 0),
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
			input: mcmsapi.MultisigConfig{
				Signers: []mcmsapi.SignerInfo{
					{SignerAddress: damltypes.TEXT("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), SignerIndex: damltypes.INT64(0), SignerGroup: damltypes.INT64(0)},
					{SignerAddress: damltypes.TEXT("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"), SignerIndex: damltypes.INT64(1), SignerGroup: damltypes.INT64(1)},
					{SignerAddress: damltypes.TEXT("0xcccccccccccccccccccccccccccccccccccccccc"), SignerIndex: damltypes.INT64(2), SignerGroup: damltypes.INT64(1)},
					{SignerAddress: damltypes.TEXT("0xdddddddddddddddddddddddddddddddddddddddd"), SignerIndex: damltypes.INT64(3), SignerGroup: damltypes.INT64(1)},
				},
				GroupQuorums: repeatInt64(1, 2),
				GroupParents: repeatInt64(0, 0),
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
			input: mcmsapi.MultisigConfig{
				Signers: []mcmsapi.SignerInfo{
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000001"), SignerIndex: damltypes.INT64(0), SignerGroup: damltypes.INT64(0)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000002"), SignerIndex: damltypes.INT64(1), SignerGroup: damltypes.INT64(1)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000003"), SignerIndex: damltypes.INT64(2), SignerGroup: damltypes.INT64(1)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000004"), SignerIndex: damltypes.INT64(3), SignerGroup: damltypes.INT64(2)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000005"), SignerIndex: damltypes.INT64(4), SignerGroup: damltypes.INT64(2)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000006"), SignerIndex: damltypes.INT64(5), SignerGroup: damltypes.INT64(3)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000007"), SignerIndex: damltypes.INT64(6), SignerGroup: damltypes.INT64(3)},
					{SignerAddress: damltypes.TEXT("0x1000000000000000000000000000000000000008"), SignerIndex: damltypes.INT64(7), SignerGroup: damltypes.INT64(3)},
				},
				GroupQuorums: repeatInt64(2, 2, 1, 2),
				GroupParents: repeatInt64(0, 0, 0, 1),
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
			input: mcmsapi.MultisigConfig{
				Signers: []mcmsapi.SignerInfo{
					{SignerAddress: damltypes.TEXT("0xdead000000000000000000000000000000000001"), SignerIndex: damltypes.INT64(0), SignerGroup: damltypes.INT64(0)},
					{SignerAddress: damltypes.TEXT("0xdead000000000000000000000000000000000002"), SignerIndex: damltypes.INT64(1), SignerGroup: damltypes.INT64(2)},
					{SignerAddress: damltypes.TEXT("0xdead000000000000000000000000000000000003"), SignerIndex: damltypes.INT64(2), SignerGroup: damltypes.INT64(2)},
				},
				GroupQuorums: repeatInt64(1, 0, 2),
				GroupParents: repeatInt64(0, 0, 0),
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
			t.Parallel()

			result, err := toConfig(tt.input)
			require.NoError(t, err, tt.description)
			require.NotNil(t, result)

			require.Equal(t, tt.expected.Quorum, result.Quorum, "quorum mismatch")
			require.Len(t, result.Signers, len(tt.expected.Signers), "signers count mismatch")

			for i, expectedSigner := range tt.expected.Signers {
				require.Equal(t, expectedSigner, result.Signers[i], "signer mismatch at index %d", i)
			}

			compareGroupSigners(t, tt.expected.GroupSigners, result.GroupSigners)
		})
	}
}

func TestToConfigErrors(t *testing.T) {
	t.Parallel()

	validQuorums := repeatInt64(1)
	validParents := repeatInt64(0)

	tests := []struct {
		name    string
		input   mcmsapi.MultisigConfig
		wantErr string
	}{
		{
			name: "signer group out of range",
			input: mcmsapi.MultisigConfig{
				Signers: []mcmsapi.SignerInfo{
					{SignerAddress: damltypes.TEXT("0x1111111111111111111111111111111111111111"), SignerGroup: damltypes.INT64(maxMCMSGroups)},
				},
				GroupQuorums: validQuorums,
				GroupParents: validParents,
			},
			wantErr: "signer group index",
		},
		{
			name: "too many group quorums",
			input: mcmsapi.MultisigConfig{
				GroupQuorums: append(validQuorums, damltypes.INT64(1)),
				GroupParents: validParents,
			},
			wantErr: "group quorums length",
		},
		{
			name: "too many group parents",
			input: mcmsapi.MultisigConfig{
				GroupQuorums: validQuorums,
				GroupParents: append(validParents, damltypes.INT64(0)),
			},
			wantErr: "group parents length",
		},
		{
			name: "parent index out of range",
			input: mcmsapi.MultisigConfig{
				Signers: []mcmsapi.SignerInfo{
					{SignerAddress: damltypes.TEXT("0x1111111111111111111111111111111111111111"), SignerGroup: damltypes.INT64(1)},
				},
				GroupQuorums: repeatInt64(0, 1),
				GroupParents: repeatInt64(0, int64(maxMCMSGroups)),
			},
			wantErr: "group parent index",
		},
		{
			name: "invalid root config",
			input: mcmsapi.MultisigConfig{
				GroupQuorums: repeatInt64(5),
				GroupParents: validParents,
			},
			wantErr: "invalid MCMS config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := toConfig(tt.input)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func repeatInt64(values ...int64) []damltypes.INT64 {
	out := make([]damltypes.INT64, maxMCMSGroups)
	for i, v := range values {
		if i >= maxMCMSGroups {
			break
		}
		out[i] = damltypes.INT64(v)
	}

	return out
}

func compareGroupSigners(t *testing.T, expected, actual []mcmstypes.Config) {
	t.Helper()

	require.Len(t, actual, len(expected), "group signers count mismatch")

	for i := range expected {
		require.Equal(t, expected[i].Quorum, actual[i].Quorum, "group %d quorum mismatch", i)
		require.Len(t, actual[i].Signers, len(expected[i].Signers), "group %d signers count mismatch", i)

		for j, expectedSigner := range expected[i].Signers {
			require.Equal(t, expectedSigner, actual[i].Signers[j], "group %d signer mismatch at index %d", i, j)
		}

		compareGroupSigners(t, expected[i].GroupSigners, actual[i].GroupSigners)
	}
}
