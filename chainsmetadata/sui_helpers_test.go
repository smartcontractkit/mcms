package chainsmetadata

import (
	"encoding/json"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func buildTestProposalWithMetadata(chainSelector types.ChainSelector, additionalFields string) *mcms.TimelockProposal {
	builder := mcms.NewTimelockProposalBuilder()
	builder.SetVersion("v1").
		SetAction(types.TimelockActionSchedule).
		SetValidUntil(4000000000).
		AddTimelockAddress(chainSelector, "0x1234567890123456789012345678901234567890").
		AddChainMetadata(chainSelector, types.ChainMetadata{
			AdditionalFields: []byte(additionalFields),
		}).
		AddOperation(types.BatchOperation{
			ChainSelector: chainSelector,
			Transactions: []types.Transaction{
				{
					To:               "0x1234567890123456789012345678901234567890",
					Data:             []byte{0x01},
					AdditionalFields: json.RawMessage(`{}`),
				},
			},
		})
	proposal, err := builder.Build()
	if err != nil {
		panic(err)
	}

	return proposal
}

func TestSuiMetadataFromProposal(t *testing.T) {
	t.Parallel()

	chainSelector := types.ChainSelector(chainsel.GETH_TESTNET.Selector)
	validMetadataJSON := `{"mcms_package_id":"0x1","role":1,"account_obj":"0x2","registry_obj":"0x3","timelock_obj":"0x4","deployer_state_obj":"0x5"}`
	invalidJSON := `{"mcms_package_id":"0x1","role":1`
	missingFieldsJSON := `{"role":1}`

	tests := []struct {
		name        string
		selector    types.ChainSelector
		proposal    *mcms.TimelockProposal
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid metadata returns success",
			selector:    chainSelector,
			proposal:    buildTestProposalWithMetadata(chainSelector, validMetadataJSON),
			expectError: false,
		},
		{
			name:        "invalid JSON returns error",
			selector:    chainSelector,
			proposal:    buildTestProposalWithMetadata(chainSelector, invalidJSON),
			expectError: true,
			errorMsg:    "error unmarshaling sui chain metadata",
		},
		{
			name:        "missing required fields returns validation error",
			selector:    chainSelector,
			proposal:    buildTestProposalWithMetadata(chainSelector, missingFieldsJSON),
			expectError: true,
			errorMsg:    "error validating sui chain metadata",
		},
		{
			name:        "empty additional fields returns unmarshaling error",
			selector:    chainSelector,
			proposal:    buildTestProposalWithMetadata(chainSelector, ""),
			expectError: true,
			errorMsg:    "error unmarshaling sui chain metadata",
		},
		{
			name:        "missing chain selector in metadata",
			selector:    types.ChainSelector(999),
			proposal:    buildTestProposalWithMetadata(chainSelector, validMetadataJSON),
			expectError: true,
			errorMsg:    "error unmarshaling sui chain metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := SuiMetadata(tt.proposal.ChainMetadata[tt.selector])

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, sui.AdditionalFieldsMetadata{}, metadata)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "0x1", metadata.McmsPackageID)
				assert.Equal(t, sui.TimelockRole(1), metadata.Role)
				assert.Equal(t, "0x2", metadata.AccountObj)
				assert.Equal(t, "0x3", metadata.RegistryObj)
				assert.Equal(t, "0x4", metadata.TimelockObj)
			}
		})
	}
}
