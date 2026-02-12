package canton

import (
	"encoding/json"
	"testing"

	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdditionalFieldsMetadata_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   AdditionalFieldsMetadata
		wantErr string
	}{
		{
			name: "valid metadata",
			input: AdditionalFieldsMetadata{
				ChainId:              1,
				MultisigId:           "test-multisig",
				PreOpCount:           0,
				PostOpCount:          5,
				OverridePreviousRoot: false,
			},
			wantErr: "",
		},
		{
			name: "valid metadata with override",
			input: AdditionalFieldsMetadata{
				ChainId:              123,
				MultisigId:           "another-multisig",
				PreOpCount:           10,
				PostOpCount:          20,
				OverridePreviousRoot: true,
			},
			wantErr: "",
		},
		{
			name: "valid metadata with same pre and post op count",
			input: AdditionalFieldsMetadata{
				ChainId:     1,
				MultisigId:  "test-multisig",
				PreOpCount:  5,
				PostOpCount: 5,
			},
			wantErr: "",
		},
		{
			name: "missing chainId",
			input: AdditionalFieldsMetadata{
				MultisigId:  "test-multisig",
				PreOpCount:  0,
				PostOpCount: 5,
			},
			wantErr: "chainId is required",
		},
		{
			name: "missing multisigId",
			input: AdditionalFieldsMetadata{
				ChainId:     1,
				PreOpCount:  0,
				PostOpCount: 5,
			},
			wantErr: "multisigId is required",
		},
		{
			name: "postOpCount less than preOpCount",
			input: AdditionalFieldsMetadata{
				ChainId:     1,
				MultisigId:  "test-multisig",
				PreOpCount:  10,
				PostOpCount: 5,
			},
			wantErr: "postOpCount must be >= preOpCount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewChainMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		preOpCount           uint64
		postOpCount          uint64
		chainId              int64
		multisigId           string
		mcmsContractID       string
		overridePreviousRoot bool
		wantErr              string
	}{
		{
			name:           "valid metadata",
			preOpCount:     0,
			postOpCount:    5,
			chainId:        1,
			multisigId:     "test-multisig",
			mcmsContractID: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
			wantErr:        "",
		},
		{
			name:                 "valid metadata with override",
			preOpCount:           10,
			postOpCount:          20,
			chainId:              123,
			multisigId:           "another-multisig",
			mcmsContractID:       "11f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
			overridePreviousRoot: true,
			wantErr:              "",
		},
		{
			name:           "missing mcmsContractID",
			preOpCount:     0,
			postOpCount:    5,
			chainId:        1,
			multisigId:     "test-multisig",
			mcmsContractID: "",
			wantErr:        "MCMS contract ID is required",
		},
		{
			name:           "missing chainId",
			preOpCount:     0,
			postOpCount:    5,
			chainId:        0,
			multisigId:     "test-multisig",
			mcmsContractID: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
			wantErr:        "chainId is required",
		},
		{
			name:           "missing multisigId",
			preOpCount:     0,
			postOpCount:    5,
			chainId:        1,
			multisigId:     "",
			mcmsContractID: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
			wantErr:        "multisigId is required",
		},
		{
			name:           "postOpCount less than preOpCount",
			preOpCount:     10,
			postOpCount:    5,
			chainId:        1,
			multisigId:     "test-multisig",
			mcmsContractID: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
			wantErr:        "postOpCount must be >= preOpCount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewChainMetadata(
				tt.preOpCount,
				tt.postOpCount,
				tt.chainId,
				tt.multisigId,
				tt.mcmsContractID,
				tt.overridePreviousRoot,
			)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, types.ChainMetadata{}, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mcmsContractID, got.MCMAddress)
				assert.Equal(t, tt.preOpCount, got.StartingOpCount)

				// Validate additional fields were marshaled correctly
				var additionalFields AdditionalFieldsMetadata
				err = json.Unmarshal(got.AdditionalFields, &additionalFields)
				require.NoError(t, err)
				assert.Equal(t, tt.chainId, additionalFields.ChainId)
				assert.Equal(t, tt.multisigId, additionalFields.MultisigId)
				assert.Equal(t, tt.preOpCount, additionalFields.PreOpCount)
				assert.Equal(t, tt.postOpCount, additionalFields.PostOpCount)
				assert.Equal(t, tt.overridePreviousRoot, additionalFields.OverridePreviousRoot)
			}
		})
	}
}

func TestValidateChainMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata types.ChainMetadata
		wantErr  string
	}{
		{
			name: "valid metadata",
			metadata: types.ChainMetadata{
				MCMAddress:      "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig",
					"preOpCount": 0,
					"postOpCount": 5,
					"overridePreviousRoot": false
				}`),
			},
			wantErr: "",
		},
		{
			name: "invalid additional fields - missing chainId",
			metadata: types.ChainMetadata{
				MCMAddress:      "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"multisigId": "test-multisig",
					"preOpCount": 0,
					"postOpCount": 5
				}`),
			},
			wantErr: "chainId is required",
		},
		{
			name: "invalid additional fields - missing multisigId",
			metadata: types.ChainMetadata{
				MCMAddress:      "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"preOpCount": 0,
					"postOpCount": 5
				}`),
			},
			wantErr: "multisigId is required",
		},
		{
			name: "invalid additional fields - postOpCount less than preOpCount",
			metadata: types.ChainMetadata{
				MCMAddress:      "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig",
					"preOpCount": 10,
					"postOpCount": 5
				}`),
			},
			wantErr: "postOpCount must be >= preOpCount",
		},
		{
			name: "invalid JSON in additional fields",
			metadata: types.ChainMetadata{
				MCMAddress:       "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
				StartingOpCount:  0,
				AdditionalFields: json.RawMessage(`{invalid json}`),
			},
			wantErr: "unable to unmarshal additional fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateChainMetadata(tt.metadata)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTimelockRole_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role TimelockRole
		want string
	}{
		{
			name: "bypasser",
			role: TimelockRoleBypasser,
			want: "Bypasser",
		},
		{
			name: "proposer",
			role: TimelockRoleProposer,
			want: "Proposer",
		},
		{
			name: "canceller",
			role: TimelockRoleCanceller,
			want: "Canceller",
		},
		{
			name: "unknown",
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
			name: "bypasser",
			role: TimelockRoleBypasser,
			want: 0,
		},
		{
			name: "canceller",
			role: TimelockRoleCanceller,
			want: 1,
		},
		{
			name: "proposer",
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
