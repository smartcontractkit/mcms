package sui

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockConfigurer(t *testing.T) {
	t.Parallel()

	mcmsPackageID := "0x1"

	configurer := NewTimelockConfigurer(mcmsPackageID)
	require.NotNil(t, configurer)
	assert.Equal(t, mcmsPackageID, configurer.mcmsPackageID)
}

func TestTimelockConfigurer_UpdateDelay(t *testing.T) {
	t.Parallel()

	mcmsPackageID := "0x1"
	validTimelockAddress := "0x1234"

	configurer := NewTimelockConfigurer(mcmsPackageID)

	tests := []struct {
		name            string
		timelockAddress string
		newDelay        uint64
		assertion       assert.ErrorAssertionFunc
		wantErr         string
	}{
		{
			name:            "success returns prepared tx with empty hash",
			timelockAddress: validTimelockAddress,
			newDelay:        3600,
			assertion:       assert.NoError,
		},
		{
			name:            "empty timelock address rejected",
			timelockAddress: "",
			newDelay:        3600,
			assertion:       assert.Error,
			wantErr:         "timelock address is required",
		},
		{
			name:            "invalid timelock address rejected",
			timelockAddress: "0xtimelock",
			newDelay:        3600,
			assertion:       assert.Error,
			wantErr:         "decoding timelock address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := configurer.UpdateDelay(t.Context(), tt.timelockAddress, tt.newDelay)

			tt.assertion(t, err)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.Empty(t, got.Hash, "Sui UpdateDelay must return empty Hash (prepared tx)")
			assert.Equal(t, chainsel.FamilySui, got.ChainFamily)

			tx, ok := got.RawData.(types.Transaction)
			require.True(t, ok, "RawData should be mcms types.Transaction")
			assert.Equal(t, mcmsPackageID, tx.To)

			var fields AdditionalFields
			require.NoError(t, json.Unmarshal(tx.AdditionalFields, &fields))
			assert.Equal(t, suiTimelockUpdateMinDelayModuleName, fields.ModuleName)
			assert.Equal(t, suiTimelockUpdateMinDelayFunctionName, fields.Function)
			assert.Equal(t, tt.timelockAddress, fields.StateObj)
			assert.NotEmpty(t, tx.Data, "BCS-encoded new delay must be present")
		})
	}
}

func TestSerializeTimelockUpdateMinDelay(t *testing.T) {
	t.Parallel()

	timelockAddress := "0x1234"
	data, err := serializeTimelockUpdateMinDelay(timelockAddress, 3600)
	require.NoError(t, err)
	require.Len(t, data, AddressLen+8)

	addr, err := AddressFromHex(timelockAddress)
	require.NoError(t, err)
	assert.Equal(t, addr.Bytes(), data[:AddressLen])
	assert.Equal(t, uint64(3600), binary.LittleEndian.Uint64(data[AddressLen:]))
}
