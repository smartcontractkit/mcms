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

	tests := []struct {
		name    string
		input   json.RawMessage
		wantErr string
	}{
		{
			name:  "empty additional fields",
			input: nil,
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
