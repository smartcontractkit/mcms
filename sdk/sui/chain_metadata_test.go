package sui

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimelockRole_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role TimelockRole
		want string
	}{
		{
			name: "bypasser role",
			role: TimelockRoleBypasser,
			want: "bypasser",
		},
		{
			name: "canceller role",
			role: TimelockRoleCanceller,
			want: "canceller",
		},
		{
			name: "proposer role",
			role: TimelockRoleProposer,
			want: "proposer",
		},
		{
			name: "unknown role",
			role: TimelockRole(99),
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.role.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockRole_Byte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role TimelockRole
		want uint8
	}{
		{
			name: "bypasser role byte",
			role: TimelockRoleBypasser,
			want: 0,
		},
		{
			name: "canceller role byte",
			role: TimelockRoleCanceller,
			want: 1,
		},
		{
			name: "proposer role byte",
			role: TimelockRoleProposer,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.role.Byte()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockRole_Constants(t *testing.T) {
	t.Parallel()

	// Test that the constants have the expected values
	assert.Equal(t, TimelockRole(0), TimelockRoleBypasser)
	assert.Equal(t, TimelockRole(1), TimelockRoleCanceller)
	assert.Equal(t, TimelockRole(2), TimelockRoleProposer)
}

func TestAdditionalFieldsMetadata_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata AdditionalFieldsMetadata
		wantJSON string
	}{
		{
			name: "bypasser role with package ID",
			metadata: AdditionalFieldsMetadata{
				Role:          TimelockRoleBypasser,
				McmsPackageID: "0x123456789abcdef",
			},
			wantJSON: `{"role":0,"mcms_package_id":"0x123456789abcdef"}`,
		},
		{
			name: "proposer role with package ID",
			metadata: AdditionalFieldsMetadata{
				Role:          TimelockRoleProposer,
				McmsPackageID: "0xfedcba9876543210",
			},
			wantJSON: `{"role":2,"mcms_package_id":"0xfedcba9876543210"}`,
		},
		{
			name: "canceller role with empty package ID",
			metadata: AdditionalFieldsMetadata{
				Role:          TimelockRoleCanceller,
				McmsPackageID: "",
			},
			wantJSON: `{"role":1,"mcms_package_id":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test marshaling
			gotJSON, err := json.Marshal(tt.metadata)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(gotJSON))

			// Test unmarshaling
			var gotMetadata AdditionalFieldsMetadata
			err = json.Unmarshal([]byte(tt.wantJSON), &gotMetadata)
			require.NoError(t, err)
			assert.Equal(t, tt.metadata, gotMetadata)
		})
	}
}

func TestAdditionalFieldsMetadata_RoundTrip(t *testing.T) {
	t.Parallel()

	original := AdditionalFieldsMetadata{
		Role:          TimelockRoleProposer,
		McmsPackageID: "0x1234567890abcdef1234567890abcdef12345678",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back to struct
	var roundTrip AdditionalFieldsMetadata
	err = json.Unmarshal(jsonData, &roundTrip)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, original, roundTrip)
	assert.Equal(t, original.Role, roundTrip.Role)
	assert.Equal(t, original.McmsPackageID, roundTrip.McmsPackageID)
}
