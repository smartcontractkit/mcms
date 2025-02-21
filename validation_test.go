package mcms

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	solana2 "github.com/gagliardetto/solana-go"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func TestValidateAdditionalFieldsMetadata(t *testing.T) {
	t.Parallel()

	// Create a valid Solana metadata instance.
	validMetadata := solana.AdditionalFieldsMetadata{
		ProposerRoleAccessController:  solana2.NewWallet().PublicKey(),
		CancellerRoleAccessController: solana2.NewWallet().PublicKey(),
		BypasserRoleAccessController:  solana2.NewWallet().PublicKey(),
	}
	validMetadataJSON, err := json.Marshal(validMetadata)
	require.NoError(t, err)

	// Create an invalid Solana metadata instance (empty public keys).
	invalidMetadata := solana.AdditionalFieldsMetadata{}
	invalidMetadataJSON, err := json.Marshal(invalidMetadata)
	require.NoError(t, err)

	tests := []struct {
		name             string
		chainSelector    types.ChainSelector
		additionalFields types.ChainMetadata
		expectedErr      error
	}{
		{
			name:             "valid Solana metadata",
			chainSelector:    types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			additionalFields: types.ChainMetadata{AdditionalFields: validMetadataJSON},
			expectedErr:      nil,
		},
		{
			name:             "invalid Solana metadata value",
			chainSelector:    types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			additionalFields: types.ChainMetadata{AdditionalFields: invalidMetadataJSON},
			expectedErr:      errors.New("Key: 'AdditionalFieldsMetadata.ProposerRoleAccessController' Error:Field validation for 'ProposerRoleAccessController' failed on the 'required' tag\nKey: 'AdditionalFieldsMetadata.CancellerRoleAccessController' Error:Field validation for 'CancellerRoleAccessController' failed on the 'required' tag\nKey: 'AdditionalFieldsMetadata.BypasserRoleAccessController' Error:Field validation for 'BypasserRoleAccessController' failed on the 'required' tag"),
		},
		{
			name:             "unknown chain family",
			chainSelector:    types.ChainSelector(999),
			additionalFields: types.ChainMetadata{AdditionalFields: nil},
			expectedErr:      errors.New("family not found for selector 999"),
		},
		{
			name:             "invalid JSON for Solana metadata",
			chainSelector:    types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			additionalFields: types.ChainMetadata{AdditionalFields: []byte("invalid JSON")},
			expectedErr:      errors.New("unable to unmarshal additional fields: invalid character 'i' looking for beginning of value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateChainMetadata(tt.additionalFields, tt.chainSelector)
			if tt.expectedErr != nil {
				require.Error(t, err)
				// Assert that the error message contains the expected error text.
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()

	validEVMFields := evm.AdditionalFields{Value: big.NewInt(100)}
	validEVMFieldsJSON, err := json.Marshal(validEVMFields)
	require.NoError(t, err)

	invalidEVMFields := evm.AdditionalFields{
		Value: big.NewInt(-100),
	}
	invalidEVMFieldsJSON, err := json.Marshal(invalidEVMFields)
	require.NoError(t, err)

	validSolanaFields := solana.AdditionalFields{Accounts: []*solana2.AccountMeta{{
		PublicKey: solana2.NewWallet().PublicKey(),
	}}}
	validSolanaFieldsJSON, err := json.Marshal(validSolanaFields)
	require.NoError(t, err)

	tests := []struct {
		name        string
		operation   types.Operation
		expectedErr error
	}{
		{
			name: "valid EVM fields",
			operation: types.Operation{
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Transaction: types.Transaction{
					AdditionalFields: validEVMFieldsJSON,
				},
			},

			expectedErr: nil,
		},
		{
			name: "invalid EVM fields",
			operation: types.Operation{
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Transaction: types.Transaction{
					AdditionalFields: invalidEVMFieldsJSON,
				},
			},
			expectedErr: errors.New("invalid EVM value"),
		},
		{
			name: "valid Solana fields",
			operation: types.Operation{
				ChainSelector: types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
				Transaction: types.Transaction{
					AdditionalFields: validSolanaFieldsJSON,
				},
			},
			expectedErr: nil,
		},
		{
			name: "unknown chain family",
			operation: types.Operation{
				ChainSelector: 999,
				Transaction: types.Transaction{
					AdditionalFields: nil,
				},
			},
			expectedErr: errors.New("family not found for selector 999"),
		},
		{
			name: "invalid JSON for EVM fields",
			operation: types.Operation{
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid JSON"),
				},
			},
			expectedErr: errors.New("failed to unmarshal EVM additional fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAdditionalFields(tt.operation.Transaction.AdditionalFields, tt.operation.ChainSelector)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
