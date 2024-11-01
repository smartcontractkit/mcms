package mcms

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: Should go to EVM SDK. This should just tests the router function actuallty routes to the correct chain
func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validEVMFields := evm.EVMAdditionalFields{
		Value: big.NewInt(100),
	}
	validEVMFieldsJSON, err := json.Marshal(validEVMFields)
	require.NoError(t, err)
	invalidEVMFields := evm.EVMAdditionalFields{
		Value: big.NewInt(-100),
	}
	invalidEVMFieldsJSON, err := json.Marshal(invalidEVMFields)
	require.NoError(t, err)

	tests := []struct {
		name        string
		operation   types.ChainOperation
		expectedErr error
	}{
		{
			name: "valid EVM fields",
			operation: types.ChainOperation{
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: types.Operation{
					AdditionalFields: validEVMFieldsJSON,
				},
			},

			expectedErr: nil,
		},
		{
			name: "invalid EVM fields",
			operation: types.ChainOperation{
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: types.Operation{
					AdditionalFields: invalidEVMFieldsJSON,
				},
			},
			expectedErr: errors.New("invalid EVM value"),
		},
		{
			name: "unknown chain family",
			operation: types.ChainOperation{
				ChainSelector: 999,
				Operation: types.Operation{
					AdditionalFields: nil,
				},
			},
			expectedErr: errors.New("family not found for selector 999"),
		},
		{
			name: "invalid JSON for EVM fields",
			operation: types.ChainOperation{
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: types.Operation{
					AdditionalFields: []byte("invalid JSON"),
				},
			},
			expectedErr: errors.New("failed to unmarshal EVM additional fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAdditionalFields(tt.operation.AdditionalFields, tt.operation.ChainSelector)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
