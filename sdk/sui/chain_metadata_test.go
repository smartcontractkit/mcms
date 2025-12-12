package sui

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
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
	assert.Equal(t, TimelockRoleBypasser, TimelockRole(0))
	assert.Equal(t, TimelockRoleCanceller, TimelockRole(1))
	assert.Equal(t, TimelockRoleProposer, TimelockRole(2))
}

func TestAdditionalFields_MetadataJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata AdditionalFieldsMetadata
		wantJSON string
	}{
		{
			name: "complete metadata with all fields",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleBypasser,
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "0xaccount123",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "0xdeployer",
			},
			wantJSON: `{"role":0,"mcms_package_id":"0x123456789abcdef","account_obj":"0xaccount123","registry_obj":"0xregistry456","timelock_obj":"0xtimelock789","deployer_state_obj":"0xdeployer"}`,
		},
		{
			name: "proposer role with all objects",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "0xfedcba9876543210",
				AccountObj:       "0xacc",
				RegistryObj:      "0xreg",
				TimelockObj:      "0xtimelock",
				DeployerStateObj: "0xdeployer",
			},
			wantJSON: `{"role":2,"mcms_package_id":"0xfedcba9876543210","account_obj":"0xacc","registry_obj":"0xreg","timelock_obj":"0xtimelock","deployer_state_obj":"0xdeployer"}`,
		},
		{
			name: "canceller role with empty values",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleCanceller,
				McmsPackageID:    "",
				AccountObj:       "",
				RegistryObj:      "",
				TimelockObj:      "",
				DeployerStateObj: "",
			},
			wantJSON: `{"role":1,"mcms_package_id":"","account_obj":"","registry_obj":"","timelock_obj":"","deployer_state_obj":""}`,
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

func TestAdditionalFields_MetadataRoundTrip(t *testing.T) {
	t.Parallel()

	original := AdditionalFieldsMetadata{
		Role:             TimelockRoleProposer,
		McmsPackageID:    "0x1234567890abcdef1234567890abcdef12345678",
		AccountObj:       "0xaccount1234567890abcdef",
		RegistryObj:      "0xregistry1234567890abcdef",
		TimelockObj:      "0xtimelock1234567890abcdef",
		DeployerStateObj: "0xdeployer1234567890abcdef",
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
	assert.Equal(t, original.AccountObj, roundTrip.AccountObj)
	assert.Equal(t, original.RegistryObj, roundTrip.RegistryObj)
	assert.Equal(t, original.TimelockObj, roundTrip.TimelockObj)
	assert.Equal(t, original.DeployerStateObj, roundTrip.DeployerStateObj)
}

func TestAdditionalFields_MetadataValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		metadata  AdditionalFieldsMetadata
		wantErr   bool
		errString string
	}{
		{
			name: "valid metadata with all fields",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "0xaccount123",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "0xdeployer123",
			},
			wantErr: false,
		},
		{
			name: "valid metadata with bypasser role",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleBypasser,
				McmsPackageID:    "0xpackage",
				AccountObj:       "0xaccount",
				RegistryObj:      "0xregistry",
				TimelockObj:      "0xtimelock",
				DeployerStateObj: "0xdeployer",
			},
			wantErr: false,
		},
		{
			name: "valid metadata with canceller role",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleCanceller,
				McmsPackageID:    "0xpackage",
				AccountObj:       "0xaccount",
				RegistryObj:      "0xregistry",
				TimelockObj:      "0xtimelock",
				DeployerStateObj: "0xdeployer",
			},
			wantErr: false,
		},
		{
			name: "invalid role too high",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRole(99),
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "0xaccount123",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "0xdeployer123",
			},
			wantErr:   true,
			errString: "invalid timelock role",
		},
		{
			name: "missing mcms package ID",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "",
				AccountObj:       "0xaccount123",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "0xdeployer123",
			},
			wantErr:   true,
			errString: "mcms package ID is required",
		},
		{
			name: "missing account object ID",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "0xdeployer123",
			},
			wantErr:   true,
			errString: "account object ID is required",
		},
		{
			name: "missing registry object ID",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "0xaccount123",
				RegistryObj:      "",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "0xdeployer123",
			},
			wantErr:   true,
			errString: "registry object ID is required",
		},
		{
			name: "missing timelock object ID",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "0xaccount123",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "",
				DeployerStateObj: "0xdeployer123",
			},
			wantErr:   true,
			errString: "timelock object ID is required",
		},
		{
			name: "missing deployer state object ID",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "0x123456789abcdef",
				AccountObj:       "0xaccount123",
				RegistryObj:      "0xregistry456",
				TimelockObj:      "0xtimelock789",
				DeployerStateObj: "",
			},
			wantErr:   true,
			errString: "deployer state object ID is required",
		},
		{
			name: "all fields empty",
			metadata: AdditionalFieldsMetadata{
				Role:             TimelockRoleProposer,
				McmsPackageID:    "",
				AccountObj:       "",
				RegistryObj:      "",
				TimelockObj:      "",
				DeployerStateObj: "",
			},
			wantErr:   true,
			errString: "mcms package ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.metadata.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewChainMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		startingOpCount  uint64
		role             TimelockRole
		mcmsPackageID    string
		mcmsObj          string
		accountObj       string
		registryObj      string
		timelockObj      string
		deployerStateObj string
		wantErr          bool
		errString        string
		expectedMeta     *types.ChainMetadata
	}{
		{
			name:             "valid chain metadata with proposer role",
			startingOpCount:  42,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "0xpackage123",
			mcmsObj:          "0xmcms123",
			accountObj:       "0xaccount123",
			registryObj:      "0xregistry123",
			timelockObj:      "0xtimelock123",
			deployerStateObj: "0xdeployer123",
			wantErr:          false,
			expectedMeta: &types.ChainMetadata{
				StartingOpCount: 42,
				MCMAddress:      "0xmcms123",
			},
		},
		{
			name:             "valid chain metadata with bypasser role",
			startingOpCount:  0,
			role:             TimelockRoleBypasser,
			mcmsPackageID:    "0xpackage456",
			mcmsObj:          "0xmcms456",
			accountObj:       "0xaccount456",
			registryObj:      "0xregistry456",
			timelockObj:      "0xtimelock456",
			deployerStateObj: "0xdeployer456",
			wantErr:          false,
			expectedMeta: &types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      "0xmcms456",
			},
		},
		{
			name:             "valid chain metadata with canceller role",
			startingOpCount:  999,
			role:             TimelockRoleCanceller,
			mcmsPackageID:    "0xpackage789",
			mcmsObj:          "0xmcms789",
			accountObj:       "0xaccount789",
			registryObj:      "0xregistry789",
			timelockObj:      "0xtimelock789",
			deployerStateObj: "0xdeployer789",
			wantErr:          false,
			expectedMeta: &types.ChainMetadata{
				StartingOpCount: 999,
				MCMAddress:      "0xmcms789",
			},
		},
		{
			name:             "empty mcms object ID",
			startingOpCount:  1,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "0xpackage123",
			mcmsObj:          "",
			accountObj:       "0xaccount123",
			registryObj:      "0xregistry123",
			timelockObj:      "0xtimelock123",
			deployerStateObj: "0xdeployer123",
			wantErr:          true,
			errString:        "mcms object ID is required",
		},
		{
			name:             "invalid role",
			startingOpCount:  1,
			role:             TimelockRole(99),
			mcmsPackageID:    "0xpackage123",
			mcmsObj:          "0xmcms123",
			accountObj:       "0xaccount123",
			registryObj:      "0xregistry123",
			timelockObj:      "0xtimelock123",
			deployerStateObj: "0xdeployer123",
			wantErr:          true,
			errString:        "additional fields are invalid: invalid timelock role",
		},
		{
			name:             "missing mcms package ID",
			startingOpCount:  1,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "",
			mcmsObj:          "0xmcms123",
			accountObj:       "0xaccount123",
			registryObj:      "0xregistry123",
			timelockObj:      "0xtimelock123",
			deployerStateObj: "0xdeployer123",
			wantErr:          true,
			errString:        "additional fields are invalid: mcms package ID is required",
		},
		{
			name:             "missing account object ID",
			startingOpCount:  1,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "0xpackage123",
			mcmsObj:          "0xmcms123",
			accountObj:       "",
			registryObj:      "0xregistry123",
			timelockObj:      "0xtimelock123",
			deployerStateObj: "0xdeployer123",
			wantErr:          true,
			errString:        "additional fields are invalid: account object ID is required",
		},
		{
			name:             "missing registry object ID",
			startingOpCount:  1,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "0xpackage123",
			mcmsObj:          "0xmcms123",
			accountObj:       "0xaccount123",
			registryObj:      "",
			timelockObj:      "0xtimelock123",
			deployerStateObj: "0xdeployer123",
			wantErr:          true,
			errString:        "additional fields are invalid: registry object ID is required",
		},
		{
			name:             "missing timelock object ID",
			startingOpCount:  1,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "0xpackage123",
			mcmsObj:          "0xmcms123",
			accountObj:       "0xaccount123",
			registryObj:      "0xregistry123",
			timelockObj:      "",
			deployerStateObj: "0xdeployer123",
			wantErr:          true,
			errString:        "additional fields are invalid: timelock object ID is required",
		},
		{
			name:             "all object IDs empty except mcms",
			startingOpCount:  1,
			role:             TimelockRoleProposer,
			mcmsPackageID:    "",
			mcmsObj:          "0xmcms123",
			accountObj:       "",
			registryObj:      "",
			timelockObj:      "",
			deployerStateObj: "",
			wantErr:          true,
			errString:        "additional fields are invalid: mcms package ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewChainMetadata(
				tt.startingOpCount,
				tt.role,
				tt.mcmsPackageID,
				tt.mcmsObj,
				tt.accountObj,
				tt.registryObj,
				tt.timelockObj,
				tt.deployerStateObj,
			)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
				assert.Equal(t, types.ChainMetadata{}, got)
			} else {
				require.NoError(t, err)

				// Check basic fields
				assert.Equal(t, tt.expectedMeta.StartingOpCount, got.StartingOpCount)
				assert.Equal(t, tt.expectedMeta.MCMAddress, got.MCMAddress)

				// Check additional fields can be unmarshaled properly
				var additionalFields AdditionalFieldsMetadata
				err = json.Unmarshal(got.AdditionalFields, &additionalFields)
				require.NoError(t, err)

				assert.Equal(t, tt.role, additionalFields.Role)
				assert.Equal(t, tt.mcmsPackageID, additionalFields.McmsPackageID)
				assert.Equal(t, tt.accountObj, additionalFields.AccountObj)
				assert.Equal(t, tt.registryObj, additionalFields.RegistryObj)
				assert.Equal(t, tt.timelockObj, additionalFields.TimelockObj)
				assert.Equal(t, tt.deployerStateObj, additionalFields.DeployerStateObj)

				// Validate the created metadata
				err = ValidateChainMetadata(got)
				require.NoError(t, err)
			}
		})
	}
}
