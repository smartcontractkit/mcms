package mcms

import (
	"encoding/json"
	"errors"
	"testing"

	solana2 "github.com/gagliardetto/solana-go"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		additionalFields json.RawMessage
		expectedErr      error
	}{
		{
			name:             "valid Solana metadata",
			chainSelector:    types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			additionalFields: validMetadataJSON,
			expectedErr:      nil,
		},
		{
			name:             "invalid Solana metadata value",
			chainSelector:    types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			additionalFields: invalidMetadataJSON,
			expectedErr:      errors.New("Key: 'AdditionalFieldsMetadata.ProposerRoleAccessController' Error:Field validation for 'ProposerRoleAccessController' failed on the 'required' tag\nKey: 'AdditionalFieldsMetadata.CancellerRoleAccessController' Error:Field validation for 'CancellerRoleAccessController' failed on the 'required' tag\nKey: 'AdditionalFieldsMetadata.BypasserRoleAccessController' Error:Field validation for 'BypasserRoleAccessController' failed on the 'required' tag"),
		},
		{
			name:             "unknown chain family",
			chainSelector:    types.ChainSelector(999),
			additionalFields: nil,
			expectedErr:      errors.New("family not found for selector 999"),
		},
		{
			name:             "invalid JSON for Solana metadata",
			chainSelector:    types.ChainSelector(cselectors.SOLANA_DEVNET.Selector),
			additionalFields: []byte("invalid JSON"),
			expectedErr:      errors.New("failed to unmarshal Solana additional fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAdditionalFieldsMetadata(tt.additionalFields, tt.chainSelector)
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
