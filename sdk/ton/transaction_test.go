package ton_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms/sdk/ton"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       json.RawMessage
		expectedErr bool
	}{
		{
			name:        "empty json",
			input:       json.RawMessage(""),
			expectedErr: false,
		},
		{
			name:        "valid json with missing fqn field",
			input:       json.RawMessage(`{"value": "100"}`),
			expectedErr: true,
		},
		{
			name:        "valid json with missing value field",
			input:       json.RawMessage(`{"contractFullyQualifiedName": "link.chain.ton.ccip.Router"}`),
			expectedErr: false,
		},
		{
			name:        "json with negative value",
			input:       json.RawMessage(`{"contractFullyQualifiedName": "link.chain.ton.ccip.Router", "value": "-10"}`),
			expectedErr: true,
		},
		{
			name:        "json with null value",
			input:       json.RawMessage(`{"contractFullyQualifiedName": "link.chain.ton.ccip.Router", "value": null}`),
			expectedErr: true,
		},
		{
			name:        "invalid json",
			input:       json.RawMessage(`{bad json}`),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ton.ValidateAdditionalFields(tt.input)
			if tt.expectedErr {
				assert.Error(t, err, "expected an error but got none", tt.name)
			} else {
				assert.NoError(t, err, "expected no error but got one", tt.name)
			}
		})
	}
}

func TestOperationFieldsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       *big.Int
		expectedErr bool
	}{
		{
			name:        "valid positive value",
			value:       big.NewInt(100),
			expectedErr: false,
		},
		{
			name:        "valid zero value",
			value:       big.NewInt(0),
			expectedErr: false,
		},
		{
			name:        "nil value",
			value:       nil,
			expectedErr: true,
		},
		{
			name:        "negative value",
			value:       big.NewInt(-10),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op := ton.AdditionalFields{
				Value:                      tt.value,
				ContractFullyQualifiedName: "link.chain.ton.ccip.Router",
			}

			err := op.Validate()

			if tt.expectedErr {
				assert.Error(t, err, "expected an error but got none", tt.name)
			} else {
				assert.NoError(t, err, "expected no error but got one", tt.name)
			}
		})
	}
}
