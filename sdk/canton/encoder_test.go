package canton

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestEncoder_ToRootMetadataDerivesPostOpCount(t *testing.T) {
	t.Parallel()

	additionalFields, err := json.Marshal(AdditionalFieldsMetadata{
		ChainId:    1,
		MultisigId: "mcms-ccip@party-proposer",
		InstanceId: "mcms-ccip",
	})
	require.NoError(t, err)

	encoder := NewEncoder(8706591216959472610, 3, true)
	fields, err := encoder.ToRootMetadata(types.ChainMetadata{
		StartingOpCount:  10,
		MCMAddress:       "0xabc",
		AdditionalFields: additionalFields,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(10), fields.PreOpCount)
	require.Equal(t, uint64(13), fields.PostOpCount)
	require.True(t, fields.OverridePreviousRoot)
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
