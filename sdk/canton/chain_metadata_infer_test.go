package canton

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

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

	enriched, err := EnsureChainMetadata(metadata, bop, 6, types.TimelockActionSchedule, true)
	require.NoError(t, err)

	var fields AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(enriched.AdditionalFields, &fields))
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, "mcms-ccip", fields.InstanceId)
	require.Equal(t, "mcms-ccip@participant1-localparty-1::1220acd0401d95915bef2f498a45cc8f3c43119dde50cf370864e9aa4eb03d817cfb-proposer", fields.MultisigId)
	require.Equal(t, uint64(1), fields.PreOpCount)
	require.Equal(t, uint64(7), fields.PostOpCount)
	require.True(t, fields.OverridePreviousRoot)
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}
