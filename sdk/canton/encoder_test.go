package canton

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestEncoder_ToRootMetadataReturnsStoredFields(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(AdditionalFieldsMetadata{
		ChainId:    1,
		MultisigId: mcmsInstanceIDCCIP + "@party-proposer",
		InstanceId: mcmsInstanceIDCCIP,
	})
	require.NoError(t, err)

	encoder := NewEncoder(8706591216959472610, 3, true)
	fields, err := encoder.ToRootMetadata(types.ChainMetadata{
		StartingOpCount:  10,
		MCMAddress:       "0xabc",
		AdditionalFields: additionalFields,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, mcmsInstanceIDCCIP+"@party-proposer", fields.MultisigId)
	require.Equal(t, mcmsInstanceIDCCIP, fields.InstanceId)
}

func TestEncoder_HashMetadataUsesStartingOpCountAndTxCount(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(AdditionalFieldsMetadata{
		ChainId:    1,
		MultisigId: "mcms@party-proposer",
	})
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		StartingOpCount:  10,
		MCMAddress:       "0xabc",
		AdditionalFields: additionalFields,
	}

	withTxCount := NewEncoder(8706591216959472610, 3, false)
	withoutTxCount := NewEncoder(8706591216959472610, 0, false)

	hashWithTx, err := withTxCount.HashMetadata(metadata)
	require.NoError(t, err)
	hashWithoutTx, err := withoutTxCount.HashMetadata(metadata)
	require.NoError(t, err)
	require.NotEqual(t, hashWithTx, hashWithoutTx)
}

func TestEncoder_HashOperationRequiresValidOperationFields(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(AdditionalFieldsMetadata{
		ChainId:    1,
		MultisigId: "mcms@party-proposer",
	})
	require.NoError(t, err)

	encoder := NewEncoder(8706591216959472610, 1, false)
	_, err = encoder.HashOperation(0, types.ChainMetadata{
		StartingOpCount:  1,
		AdditionalFields: additionalFields,
	}, types.Operation{
		Transaction: types.Transaction{AdditionalFields: []byte(`not-json`)},
	})
	require.ErrorContains(t, err, "unmarshal operation additional fields")
}

func TestEncoder_IntToHexPanicsOnNegative(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() { intToHex(-1) })
}

func TestEncoder_ToRootMetadataRequiresAdditionalFields(t *testing.T) {
	t.Parallel()

	encoder := NewEncoder(8706591216959472610, 1, false)
	_, err := encoder.ToRootMetadata(types.ChainMetadata{
		StartingOpCount: 1,
		MCMAddress:      "0xabc",
	})
	require.Error(t, err)
}
