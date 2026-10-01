package aptos

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

func TestRoleMetadataDeriver_DeriveRoleMetadata(t *testing.T) {
	t.Parallel()

	sourceFields := func(t *testing.T, mcmsType MCMSType) json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleProposer, MCMSType: mcmsType})
		require.NoError(t, err)

		return raw
	}
	failingResolver := sdk.RoleAddressResolverFunc(func(context.Context, sdk.RoleAddressRequest) (string, error) {
		t.Fatal("aptos must not consult the role address resolver")
		return "", nil
	})

	tests := []struct {
		name     string
		action   types.TimelockAction
		mcmsType MCMSType
		fields   json.RawMessage
		wantRole TimelockRole
		wantErr  string
	}{
		{name: "cancel", action: types.TimelockActionCancel, wantRole: TimelockRoleCanceller},
		{name: "bypass", action: types.TimelockActionBypass, wantRole: TimelockRoleBypasser},
		{name: "schedule", action: types.TimelockActionSchedule, wantRole: TimelockRoleProposer},
		{name: "cancel curse keeps mcms type", action: types.TimelockActionCancel, mcmsType: MCMSTypeCurse, wantRole: TimelockRoleCanceller},
		{name: "missing additional fields", action: types.TimelockActionCancel, fields: json.RawMessage{}, wantErr: "missing aptos additional fields"},
		{name: "invalid additional fields", action: types.TimelockActionCancel, fields: json.RawMessage(`{`), wantErr: "unable to unmarshal aptos additional fields"},
		{name: "unknown action", action: types.TimelockAction("unknown"), wantErr: "failed to resolve aptos role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fields := tt.fields
			if fields == nil {
				fields = sourceFields(t, tt.mcmsType)
			}
			source := types.ChainMetadata{StartingOpCount: 5, MCMAddress: "0x1", AdditionalFields: fields}

			got, err := NewRoleMetadataDeriver().DeriveRoleMetadata(t.Context(), 1, source, tt.action, failingResolver)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, source.MCMAddress, got.MCMAddress)
			require.Equal(t, source.StartingOpCount, got.StartingOpCount)

			var derived AdditionalFieldsMetadata
			require.NoError(t, json.Unmarshal(got.AdditionalFields, &derived))
			require.Equal(t, tt.wantRole, derived.Role)
			require.Equal(t, tt.mcmsType, derived.MCMSType)
		})
	}
}
