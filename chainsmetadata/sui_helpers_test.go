package chainsmetadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func TestSuiMetadata(t *testing.T) {
	t.Parallel()

	validMetadata := types.ChainMetadata{
		AdditionalFields: []byte(`{"mcms_package_id":"0x1","role":1,"account_obj":"0x2","registry_obj":"0x3","timelock_obj":"0x4","deployer_state_obj":"0x5"}`),
	}

	tests := []struct {
		name        string
		metadata    types.ChainMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid metadata returns success",
			metadata: validMetadata,
		},
		{
			name: "invalid JSON returns error",
			metadata: types.ChainMetadata{
				AdditionalFields: []byte(`{"mcms_package_id":"0x1","role":1`),
			},
			expectError: true,
			errorMsg:    "error unmarshaling sui chain metadata",
		},
		{
			name: "missing required fields returns validation error",
			metadata: types.ChainMetadata{
				AdditionalFields: []byte(`{"role":1}`),
			},
			expectError: true,
			errorMsg:    "error validating sui chain metadata",
		},
		{
			name: "empty additional fields returns unmarshaling error",
			metadata: types.ChainMetadata{
				AdditionalFields: nil,
			},
			expectError: true,
			errorMsg:    "error unmarshaling sui chain metadata",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := SuiMetadata(tt.metadata)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, sui.AdditionalFieldsMetadata{}, metadata)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, "0x1", metadata.McmsPackageID)
			assert.Equal(t, sui.TimelockRole(1), metadata.Role)
			assert.Equal(t, "0x2", metadata.AccountObj)
			assert.Equal(t, "0x3", metadata.RegistryObj)
			assert.Equal(t, "0x4", metadata.TimelockObj)
		})
	}
}
