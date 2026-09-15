package stellar_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/stellar"
	"github.com/smartcontractkit/mcms/types"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		additionalFields json.RawMessage
		expectErr        string
	}{
		{
			name:             "valid",
			additionalFields: json.RawMessage(`{"family":"stellar","encodingVersion":1}`),
		},
		{
			name:             "empty",
			additionalFields: json.RawMessage(``),
			expectErr:        "missing Stellar transaction additional fields",
		},
		{
			name:             "empty object",
			additionalFields: json.RawMessage(`{}`),
			expectErr:        "invalid Stellar transaction family",
		},
		{
			name:             "wrong family",
			additionalFields: json.RawMessage(`{"family":"evm","encodingVersion":1}`),
			expectErr:        "invalid Stellar transaction family",
		},
		{
			name:             "wrong encoding version",
			additionalFields: json.RawMessage(`{"family":"stellar","encodingVersion":2}`),
			expectErr:        "unsupported",
		},
		{
			name:             "missing encoding version",
			additionalFields: json.RawMessage(`{"family":"stellar"}`),
			expectErr:        "missing Stellar transaction encodingVersion",
		},
		{
			name:             "unknown key",
			additionalFields: json.RawMessage(`{"family":"stellar","encodingVersion":1,"target":"forbidden"}`),
			expectErr:        "unknown field",
		},
		{
			name:             "trailing JSON",
			additionalFields: json.RawMessage(`{"family":"stellar","encodingVersion":1} null`),
			expectErr:        "multiple JSON values",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := stellar.ValidateAdditionalFields(tc.additionalFields)
			if tc.expectErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectErr)
			}
		})
	}
}

func TestValidateChainMetadata(t *testing.T) {
	t.Parallel()

	address := testContractID(t, 130)

	tests := []struct {
		name      string
		metadata  types.ChainMetadata
		expectErr string
	}{
		{
			name: "valid with both fields",
			metadata: types.ChainMetadata{
				MCMAddress:       address,
				AdditionalFields: json.RawMessage(`{"configVersion":1,"encodingVersion":1}`),
			},
		},
		{
			name: "valid with empty additional fields",
			metadata: types.ChainMetadata{
				MCMAddress: address,
			},
		},
		{
			name: "valid with config version only",
			metadata: types.ChainMetadata{
				MCMAddress:       address,
				AdditionalFields: json.RawMessage(`{"configVersion":2}`),
			},
		},
		{
			name: "empty address",
			metadata: types.ChainMetadata{
				AdditionalFields: json.RawMessage(`{"configVersion":1}`),
			},
			expectErr: "invalid Stellar MCMAddress",
		},
		{
			name: "malformed strkey address",
			metadata: types.ChainMetadata{
				MCMAddress:       "CNotARealContractID",
				AdditionalFields: json.RawMessage(`{"configVersion":1}`),
			},
			expectErr: "invalid Stellar MCMAddress",
		},
		{
			name: "wrong encoding version",
			metadata: types.ChainMetadata{
				MCMAddress:       address,
				AdditionalFields: json.RawMessage(`{"configVersion":1,"encodingVersion":2}`),
			},
			expectErr: "unsupported",
		},
		{
			name: "unknown key",
			metadata: types.ChainMetadata{
				MCMAddress:       address,
				AdditionalFields: json.RawMessage(`{"configVersion":1,"target":"forbidden"}`),
			},
			expectErr: "unknown field",
		},
		{
			name: "trailing JSON",
			metadata: types.ChainMetadata{
				MCMAddress:       address,
				AdditionalFields: json.RawMessage(`{"configVersion":1} null`),
			},
			expectErr: "multiple JSON values",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := stellar.ValidateChainMetadata(tc.metadata)
			if tc.expectErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectErr)
			}
		})
	}
}
