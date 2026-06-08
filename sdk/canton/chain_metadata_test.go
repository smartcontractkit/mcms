package canton

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestCantonRoleFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		action       types.TimelockAction
		expectedRole TimelockRole
		expectError  bool
	}{
		{
			name:         "bypass action returns bypasser role",
			action:       types.TimelockActionBypass,
			expectedRole: TimelockRoleBypasser,
		},
		{
			name:         "schedule action returns proposer role",
			action:       types.TimelockActionSchedule,
			expectedRole: TimelockRoleProposer,
		},
		{
			name:         "cancel action returns canceller role",
			action:       types.TimelockActionCancel,
			expectedRole: TimelockRoleCanceller,
		},
		{
			name:        "unknown action returns error",
			action:      types.TimelockAction("unknown"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := CantonRoleFromAction(tt.action)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedRole, role)
		})
	}
}

func TestAdditionalFieldsMetadataValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (AdditionalFieldsMetadata{
		ChainId:    1,
		MultisigId: "mcms@party-proposer",
	}).Validate())

	require.Error(t, (AdditionalFieldsMetadata{MultisigId: "x"}).Validate())
	require.Error(t, (AdditionalFieldsMetadata{ChainId: -1, MultisigId: "x"}).Validate())
}

func TestNewChainMetadata(t *testing.T) {
	t.Parallel()

	addr := "0xd4dcbc33d025740c32b65cb60d208a7eb8f99b3d90903ffe52616e14f9096995"
	meta, err := NewChainMetadata(5, 1, "mcms@party-proposer", addr, "mcms")
	require.NoError(t, err)
	require.Equal(t, uint64(5), meta.StartingOpCount)
	require.Equal(t, addr, meta.MCMAddress)

	var fields AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(meta.AdditionalFields, &fields))
	require.Equal(t, int64(1), fields.ChainId)
	require.Equal(t, "mcms", fields.InstanceId)
}

func TestValidateChainMetadata(t *testing.T) {
	t.Parallel()

	addr := "0xd4dcbc33d025740c32b65cb60d208a7eb8f99b3d90903ffe52616e14f9096995"
	meta, err := NewChainMetadata(0, 1, "mcms@party-proposer", addr, "mcms")
	require.NoError(t, err)
	require.NoError(t, ValidateChainMetadata(meta))

	meta.AdditionalFields = []byte(`{invalid`)
	require.Error(t, ValidateChainMetadata(meta))
}

func TestNewChainMetadataErrors(t *testing.T) {
	t.Parallel()

	_, err := NewChainMetadata(0, 1, "id", "", "mcms")
	require.ErrorContains(t, err, "InstanceAddress is required")

	_, err = NewChainMetadata(0, 1, "id", "0xshort", "mcms")
	require.ErrorContains(t, err, "64 characters")

	_, err = NewChainMetadata(0, 0, "id", validTestInstanceAddress(t), "mcms")
	require.ErrorContains(t, err, "chainId must be positive")
}

func validTestInstanceAddress(t *testing.T) string {
	t.Helper()

	return "0xd4dcbc33d025740c32b65cb60d208a7eb8f99b3d90903ffe52616e14f9096995"
}
