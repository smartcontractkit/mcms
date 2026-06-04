package canton

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validFields, err := json.Marshal(AdditionalFields{
		TargetInstanceAddress: "counter@party::abc",
		FunctionName:          "Increment",
		OperationData:         "deadbeef",
	})
	require.NoError(t, err)

	invalidAddressFields, err := json.Marshal(AdditionalFields{
		TargetInstanceAddress: "invalid-address",
		FunctionName:          "Increment",
	})
	require.NoError(t, err)

	invalidOpDataFields, err := json.Marshal(AdditionalFields{
		TargetInstanceAddress: "counter@party::abc",
		FunctionName:          "Increment",
		OperationData:         "not-hex",
	})
	require.NoError(t, err)

	onlyFunctionNameFields, err := json.Marshal(AdditionalFields{
		FunctionName: "Increment",
	})
	require.NoError(t, err)

	opDataWith0xPrefix, err := json.Marshal(AdditionalFields{
		TargetInstanceAddress: "counter@party::abc",
		FunctionName:          "Increment",
		OperationData:         "0xdeadbeef",
	})
	require.NoError(t, err)

	onlyOpDataFields, err := json.Marshal(AdditionalFields{
		OperationData: "deadbeef",
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   json.RawMessage
		wantErr string
	}{
		{
			name:    "empty additional fields",
			input:   nil,
			wantErr: "Canton additional fields are required",
		},
		{
			name:  "valid additional fields",
			input: validFields,
		},
		{
			name:    "invalid target instance address",
			input:   invalidAddressFields,
			wantErr: "targetInstanceAddress must be in instanceId@partyId format",
		},
		{
			name:    "invalid operation data",
			input:   invalidOpDataFields,
			wantErr: "operationData must be hex-encoded",
		},
		{
			name:    "function name without target instance address",
			input:   onlyFunctionNameFields,
			wantErr: "targetInstanceAddress is required when functionName is set",
		},
		{
			name:    "operation data with 0x prefix",
			input:   opDataWith0xPrefix,
			wantErr: "operationData must be hex-encoded without 0x prefix",
		},
		{
			name:    "operation data without target and function",
			input:   onlyOpDataFields,
			wantErr: "targetInstanceAddress is required when operationData is set",
		},
		{
			name:    "invalid JSON",
			input:   []byte("invalid JSON"),
			wantErr: "failed to unmarshal Canton additional fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAdditionalFields(tt.input)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
