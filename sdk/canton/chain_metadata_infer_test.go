package canton

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestResolveAdditionalFieldsMetadataUsesStoredFields(t *testing.T) {
	t.Parallel()

	staleFields := mustJSON(t, AdditionalFieldsMetadata{
		ChainId:    1,
		MultisigId: mcmsInstanceIDCCIP + "@party-proposer",
		InstanceId: mcmsInstanceIDCCIP,
	})

	metadata := types.ChainMetadata{
		StartingOpCount:  4,
		MCMAddress:       "0xabc",
		AdditionalFields: staleFields,
	}

	fields, err := resolveAdditionalFieldsMetadata(metadata, types.BatchOperation{}, types.TimelockActionSchedule)
	require.NoError(t, err)
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, mcmsInstanceIDCCIP, fields.InstanceId)
}

func TestEnsureChainMetadataInfersMissingAdditionalFields(t *testing.T) {
	t.Parallel()

	bop := types.BatchOperation{
		ChainSelector: 8706591216959472610,
		Transactions: []types.Transaction{{
			AdditionalFields: mustJSON(t, AdditionalFields{
				TargetInstanceAddress: "globalconfig-rklfx@participant1-localparty-1::1220acd0401d95915bef2f498a45cc8f3c43119dde50cf370864e9aa4eb03d817cfb",
				FunctionName:          "ApplyDestChainConfigUpdates",
			}),
		}},
	}

	metadata := types.ChainMetadata{
		StartingOpCount: 1,
		MCMAddress:      "0xd4dcbc33d025740c32b65cb60d208a7eb8f99b3d90903ffe52616e14f9096995",
	}

	enriched, err := EnsureChainMetadata(metadata, bop, types.TimelockActionSchedule)
	require.NoError(t, err)

	var fields AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(enriched.AdditionalFields, &fields))
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, mcmsInstanceIDCCIP, fields.InstanceId)
	require.Equal(t, mcmsInstanceIDCCIP+"@participant1-localparty-1::1220acd0401d95915bef2f498a45cc8f3c43119dde50cf370864e9aa4eb03d817cfb-proposer", fields.MultisigId)
	require.Equal(t, uint64(1), enriched.StartingOpCount)
}

func TestResolveAdditionalFieldsMetadataErrors(t *testing.T) {
	t.Parallel()

	_, err := resolveAdditionalFieldsMetadata(types.ChainMetadata{
		MCMAddress: "0xabc",
	}, types.BatchOperation{}, types.TimelockActionSchedule)
	require.ErrorContains(t, err, "unable to infer Canton party")

	_, err = resolveAdditionalFieldsMetadata(types.ChainMetadata{
		MCMAddress: "0xnotamatch",
	}, types.BatchOperation{
		Transactions: []types.Transaction{{
			To: "target@participant1-localparty-1::1220acd0401d95915bef2f498a45cc8f3c43119dde50cf370864e9aa4eb03d817cfb",
		}},
	}, types.TimelockActionSchedule)
	require.ErrorContains(t, err, "unable to infer MCMS instanceId")

	_, err = resolveAdditionalFieldsMetadata(types.ChainMetadata{
		AdditionalFields: []byte(`{invalid`),
	}, types.BatchOperation{}, types.TimelockActionSchedule)
	require.ErrorContains(t, err, "unmarshal metadata additional fields")
}

func TestEnsureChainMetadataBypasserRole(t *testing.T) {
	t.Parallel()

	bop := types.BatchOperation{
		Transactions: []types.Transaction{{
			To: "counter@" + testParty,
		}},
	}
	metadata := types.ChainMetadata{
		MCMAddress: "0xd4dcbc33d025740c32b65cb60d208a7eb8f99b3d90903ffe52616e14f9096995",
	}

	enriched, err := EnsureChainMetadata(metadata, bop, types.TimelockActionBypass)
	require.NoError(t, err)

	var fields AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(enriched.AdditionalFields, &fields))
	require.Contains(t, fields.MultisigId, "-bypasser")
}

func TestPartyFromRawInstanceAddress(t *testing.T) {
	t.Parallel()

	require.Equal(t, testParty, partyFromRawInstanceAddress("mcms-ccip@"+testParty))
	require.Empty(t, partyFromRawInstanceAddress("noseparator"))
	require.Empty(t, partyFromRawInstanceAddress("@onlyparty"))
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}
