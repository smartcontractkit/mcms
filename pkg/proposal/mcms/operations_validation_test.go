package mcms

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms/sdk/evm"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validEVMFields := evm.OperationFieldsEVM{
		Value: big.NewInt(100),
	}
	validEVMFieldsJSON, err := json.Marshal(validEVMFields)
	require.NoError(t, err)
	invalidEVMFields := evm.OperationFieldsEVM{
		Value: big.NewInt(-100),
	}
	invalidEVMFieldsJSON, err := json.Marshal(invalidEVMFields)
	require.NoError(t, err)

	tests := []struct {
		name        string
		operation   ChainOperation
		expectedErr error
	}{
		{
			name: "valid EVM fields",
			operation: ChainOperation{
				ChainIdentifier: ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: Operation{
					AdditionalFields: validEVMFieldsJSON,
				},
			},

			expectedErr: nil,
		},
		{
			name: "invalid EVM fields",
			operation: ChainOperation{
				ChainIdentifier: ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: Operation{
					AdditionalFields: invalidEVMFieldsJSON,
				},
			},
			expectedErr: errors.New("invalid EVM value"),
		},
		{
			name: "unknown chain family",
			operation: ChainOperation{
				ChainIdentifier: 999,
				Operation: Operation{
					AdditionalFields: nil,
				},
			},
			expectedErr: errors.New("family not found for selector 999"),
		},
		{
			name: "invalid JSON for EVM fields",
			operation: ChainOperation{
				ChainIdentifier: ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: Operation{
					AdditionalFields: []byte("invalid JSON"),
				},
			},
			expectedErr: errors.New("failed to unmarshal EVM additional fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAdditionalFields(tt.operation.AdditionalFields, tt.operation.ChainIdentifier)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
