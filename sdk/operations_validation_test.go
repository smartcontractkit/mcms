package sdk

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/stretchr/testify/assert"
)

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validEVMFields := evm.OperationFieldsEVM{
		Value: big.NewInt(100),
	}
	invalidEVMFields := evm.OperationFieldsEVM{
		Value: big.NewInt(-100),
	}

	tests := []struct {
		name        string
		operation   mcms.ChainOperation
		expectedErr error
	}{
		{
			name: "valid EVM fields",
			operation: mcms.ChainOperation{
				ChainIdentifier: mcms.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: mcms.Operation{
					AdditionalFields: marshalJSON(validEVMFields),
				},
			},

			expectedErr: nil,
		},
		{
			name: "invalid EVM fields",
			operation: mcms.ChainOperation{
				ChainIdentifier: mcms.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Operation: mcms.Operation{
					AdditionalFields: marshalJSON(invalidEVMFields),
				},
			},
			expectedErr: errors.New("invalid EVM value"),
		},
		{
			name: "unknown chain family",
			operation: mcms.ChainOperation{
				ChainIdentifier: 999,
				Operation: mcms.Operation{
					AdditionalFields: nil,
				},
			},
			expectedErr: errors.New("family not found for selector 999"),
		},
		{
			name: "invalid JSON for EVM fields",
			operation: mcms.ChainOperation{
				ChainIdentifier: mcms.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
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

			err := ValidateAdditionalFields(tt.operation)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func marshalJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
