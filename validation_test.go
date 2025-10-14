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

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func TestValidateChainMetadata(t *testing.T) {
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

	// Create a valid Aptos metadata instance.
	validAptosMetadata := aptos.AdditionalFieldsMetadata{
		Role: aptos.TimelockRoleProposer,
	}
	validAptosMetadataJSON, err := json.Marshal(validAptosMetadata)
	require.NoError(t, err)

	// Create a valid Sui metadata instance.
	validSuiMetadata := sui.AdditionalFieldsMetadata{
		Role:          sui.TimelockRoleProposer,
		McmsPackageID: "0x123456789abcdef",
		AccountObj:    "0xacc",
		RegistryObj:   "0xreg",
		TimelockObj:   "0xtimelock",
	}
	validSuiMetadataJSON, err := json.Marshal(validSuiMetadata)
	require.NoError(t, err)

	// Create an invalid Sui metadata instance (empty package ID).
	invalidSuiMetadata := sui.AdditionalFieldsMetadata{
		Role:          sui.TimelockRoleProposer,
		McmsPackageID: "",
	}
	invalidSuiMetadataJSON, err := json.Marshal(invalidSuiMetadata)
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
			name:             "valid Aptos metadata",
			chainSelector:    chaintest.Chain5Selector,
			additionalFields: types.ChainMetadata{AdditionalFields: validAptosMetadataJSON},
			expectedErr:      nil,
		},
		{
			name:             "valid Sui metadata",
			chainSelector:    chaintest.Chain6Selector,
			additionalFields: types.ChainMetadata{AdditionalFields: validSuiMetadataJSON},
			expectedErr:      nil,
		},
		{
			name:             "invalid Sui metadata value",
			chainSelector:    chaintest.Chain6Selector,
			additionalFields: types.ChainMetadata{AdditionalFields: invalidSuiMetadataJSON},
			expectedErr:      errors.New("mcms package ID is required"),
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
		{
			name:             "invalid JSON for Sui metadata",
			chainSelector:    chaintest.Chain6Selector,
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

	validAptosFields := aptos.AdditionalFields{
		PackageName: "test_package",
		ModuleName:  "test_module",
		Function:    "test_function",
	}
	validAptosFieldsJSON, err := json.Marshal(validAptosFields)
	require.NoError(t, err)

	invalidAptosFields := aptos.AdditionalFields{
		PackageName: "", // Empty package name should be invalid
		ModuleName:  "test_module",
		Function:    "test_function",
	}
	invalidAptosFieldsJSON, err := json.Marshal(invalidAptosFields)
	require.NoError(t, err)

	validSuiFields := sui.AdditionalFields{
		ModuleName: "test_module",
		Function:   "test_function",
	}
	validSuiFieldsJSON, err := json.Marshal(validSuiFields)
	require.NoError(t, err)

	invalidSuiFields := sui.AdditionalFields{
		ModuleName: "", // Empty module name should be invalid
		Function:   "test_function",
	}
	invalidSuiFieldsJSON, err := json.Marshal(invalidSuiFields)
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
			name: "valid Aptos fields",
			operation: types.Operation{
				ChainSelector: chaintest.Chain5Selector,
				Transaction: types.Transaction{
					AdditionalFields: validAptosFieldsJSON,
				},
			},
			expectedErr: nil,
		},
		{
			name: "invalid Aptos fields",
			operation: types.Operation{
				ChainSelector: chaintest.Chain5Selector,
				Transaction: types.Transaction{
					AdditionalFields: invalidAptosFieldsJSON,
				},
			},
			expectedErr: errors.New("package name is required"),
		},
		{
			name: "valid Sui fields",
			operation: types.Operation{
				ChainSelector: chaintest.Chain6Selector,
				Transaction: types.Transaction{
					AdditionalFields: validSuiFieldsJSON,
				},
			},
			expectedErr: nil,
		},
		{
			name: "invalid Sui fields",
			operation: types.Operation{
				ChainSelector: chaintest.Chain6Selector,
				Transaction: types.Transaction{
					AdditionalFields: invalidSuiFieldsJSON,
				},
			},
			expectedErr: errors.New("module name length must be between 1 and 64 characters"),
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
		{
			name: "invalid JSON for Aptos fields",
			operation: types.Operation{
				ChainSelector: chaintest.Chain5Selector,
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid JSON"),
				},
			},
			expectedErr: errors.New("failed to unmarshal Aptos additional fields"),
		},
		{
			name: "invalid JSON for Sui fields",
			operation: types.Operation{
				ChainSelector: chaintest.Chain6Selector,
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid JSON"),
				},
			},
			expectedErr: errors.New("failed to unmarshal Sui additional fields"),
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
