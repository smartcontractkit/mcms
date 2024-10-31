package evm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperationFieldsEVM_Validate(t *testing.T) {
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

			op := EVMAdditionalFields{
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
