package aptos

import (
	"encoding/json"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockConfigurer(t *testing.T) {
	t.Parallel()

	configurer := NewTimelockConfigurer(nil)

	require.NotNil(t, configurer)
	require.Equal(t, MCMSTypeRegular, configurer.mcmsType)
}

func TestNewTimelockConfigurerWithMCMSType(t *testing.T) {
	t.Parallel()

	configurer := NewTimelockConfigurerWithMCMSType(nil, MCMSTypeCurse)

	require.NotNil(t, configurer)
	require.Equal(t, MCMSTypeCurse, configurer.mcmsType)
}

func TestTimelockConfigurer_UpdateDelay(t *testing.T) {
	t.Parallel()

	validMCMSAddr := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	tests := []struct {
		name         string
		configurer   *TimelockConfigurer
		mcmsAddr     string
		newDelay     uint64
		assertion    assert.ErrorAssertionFunc
		wantErr      string
		wantPackage  string
		wantModule   string
		wantFunction string
	}{
		{
			name:         "success returns prepared tx with empty hash for regular mcms",
			configurer:   NewTimelockConfigurer(nil),
			mcmsAddr:     validMCMSAddr,
			newDelay:     3600,
			assertion:    assert.NoError,
			wantPackage:  "mcms",
			wantModule:   "mcms",
			wantFunction: "timelock_update_min_delay",
		},
		{
			name:         "success returns prepared tx with empty hash for curse mcms",
			configurer:   NewTimelockConfigurerWithMCMSType(nil, MCMSTypeCurse),
			mcmsAddr:     validMCMSAddr,
			newDelay:     3600,
			assertion:    assert.NoError,
			wantPackage:  "curse_mcms",
			wantModule:   "curse_mcms",
			wantFunction: "timelock_update_min_delay",
		},
		{
			name:       "invalid mcms address rejected",
			configurer: NewTimelockConfigurer(nil),
			mcmsAddr:   "not-an-address",
			newDelay:   3600,
			assertion:  assert.Error,
			wantErr:    "failed to parse MCMS address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.configurer.UpdateDelay(t.Context(), tt.mcmsAddr, tt.newDelay)

			tt.assertion(t, err)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.Empty(t, got.Hash, "Aptos UpdateDelay must return empty Hash (prepared tx)")
			assert.Equal(t, chainsel.FamilyAptos, got.ChainFamily)

			tx, ok := got.RawData.(types.Transaction)
			require.True(t, ok, "RawData should be mcms types.Transaction")

			var fields AdditionalFields
			require.NoError(t, json.Unmarshal(tx.AdditionalFields, &fields))
			assert.Equal(t, tt.wantPackage, fields.PackageName)
			assert.Equal(t, tt.wantModule, fields.ModuleName)
			assert.Equal(t, tt.wantFunction, fields.Function)
			assert.NotEmpty(t, tx.Data, "BCS-encoded new delay must be present")
		})
	}
}
