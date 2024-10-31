package mcms

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"

	"github.com/stretchr/testify/require"

	evm_mcms "github.com/smartcontractkit/mcms/sdk/evm/proposal/mcms"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
)

// TODO: Should go to EVM SDK. This should just tests the router function actuallty routes to the correct chain
func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validEVMFields := evm_mcms.EVMAdditionalFields{
		Value: big.NewInt(100),
	}
	validEVMFieldsJSON, err := json.Marshal(validEVMFields)
	require.NoError(t, err)
	invalidEVMFields := evm_mcms.EVMAdditionalFields{
		Value: big.NewInt(-100),
	}
	invalidEVMFieldsJSON, err := json.Marshal(invalidEVMFields)
	require.NoError(t, err)

	tests := []struct {
		name        string
		operation   mcms.ChainOperation
		expectedErr error
	}{
		{
			name: "valid EVM fields",
			operation: mcms.ChainOperation{
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: mcms.Operation{
					AdditionalFields: validEVMFieldsJSON,
				},
			},

			expectedErr: nil,
		},
		{
			name: "invalid EVM fields",
			operation: mcms.ChainOperation{
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: mcms.Operation{
					AdditionalFields: invalidEVMFieldsJSON,
				},
			},
			expectedErr: errors.New("invalid EVM value"),
		},
		{
			name: "unknown chain family",
			operation: mcms.ChainOperation{
				ChainSelector: 999,
				Operation: mcms.Operation{
					AdditionalFields: nil,
				},
			},
			expectedErr: errors.New("family not found for selector 999"),
		},
		{
			name: "invalid JSON for EVM fields",
			operation: mcms.ChainOperation{
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: mcms.Operation{
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
