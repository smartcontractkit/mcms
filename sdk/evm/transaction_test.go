package evm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAdditionalFields(t *testing.T) {
	tests := []struct {
		name        string
		input       json.RawMessage
		expectedErr bool
	}{
		{
			name:        "empty json defaults to value 0",
			input:       json.RawMessage(""),
			expectedErr: false,
		},
		{
			name:        "valid json with missing value field",
			input:       json.RawMessage(`{}`),
			expectedErr: false,
		},
		{
			name:        "json with negative value",
			input:       json.RawMessage(`{"value": "-10"}`),
			expectedErr: true,
		},
		{
			name:        "json with null value",
			input:       json.RawMessage(`{"value": null}`),
			expectedErr: true,
		},
		{
			name:        "invalid json",
			input:       json.RawMessage(`{bad json}`),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAdditionalFields(tt.input)
			if tt.expectedErr {
				assert.Error(t, err, "expected an error but got none")
			} else {
				assert.NoError(t, err, "expected no error but got one")
			}
		})
	}
}
func TestOperationFields_Validate(t *testing.T) {
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

			op := AdditionalFields{
				Value: tt.value,
			}

			err := op.Validate()

			if tt.expectedErr {
				assert.Error(t, err, "expected an error but got none")
			} else {
				assert.NoError(t, err, "expected no error but got one")
			}
		})
	}
}
