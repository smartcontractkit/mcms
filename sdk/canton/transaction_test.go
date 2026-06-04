package canton

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validFields, err := json.Marshal(AdditionalFields{
		TargetInstanceAddress: "counter@party::abc",
		FunctionName:          "Increment",
	})
	require.NoError(t, err)

	invalidAddressFields, err := json.Marshal(AdditionalFields{
		TargetInstanceAddress: "invalid-address",
		FunctionName:          "Increment",
	})
	require.NoError(t, err)

	onlyFunctionNameFields, err := json.Marshal(AdditionalFields{
		FunctionName: "Increment",
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
			wantErr: "canton additional fields are required",
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
			name:    "function name without target instance address",
			input:   onlyFunctionNameFields,
			wantErr: "targetInstanceAddress is required when functionName is set",
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

func TestOperationDataHex(t *testing.T) {
	t.Parallel()

	t.Run("encodes bytes to hex", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "deadbeef", operationDataHex([]byte{0xde, 0xad, 0xbe, 0xef}))
	})

	t.Run("empty data returns empty string", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, operationDataHex(nil))
	})

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		raw, err := hex.DecodeString("0000000000000005")
		require.NoError(t, err)
		require.Equal(t, "0000000000000005", operationDataHex(raw))
	})
}
