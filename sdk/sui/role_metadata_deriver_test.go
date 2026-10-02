package sui

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

	source, err := NewChainMetadata(5, TimelockRoleProposer, "0xpkg", "0xmcms", "0xaccount", "0xregistry", "0xtimelock", "0xdeployer")
	require.NoError(t, err)

	failingResolver := sdk.RoleAddressResolverFunc(func(context.Context, sdk.RoleAddressRequest) (string, error) {
		t.Fatal("sui must not consult the role address resolver")
		return "", nil
	})

	tests := []struct {
		name     string
		action   types.TimelockAction
		fields   json.RawMessage
		wantRole TimelockRole
		wantErr  string
	}{
		{name: "cancel", action: types.TimelockActionCancel, wantRole: TimelockRoleCanceller},
		{name: "bypass", action: types.TimelockActionBypass, wantRole: TimelockRoleBypasser},
		{name: "schedule", action: types.TimelockActionSchedule, wantRole: TimelockRoleProposer},
		{name: "missing additional fields", action: types.TimelockActionCancel, fields: json.RawMessage{}, wantErr: "missing sui additional fields"},
		{name: "invalid additional fields", action: types.TimelockActionCancel, fields: json.RawMessage(`{`), wantErr: "unable to unmarshal sui additional fields"},
		{name: "unknown action", action: types.TimelockAction("unknown"), wantErr: "failed to resolve sui role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := source
			if tt.fields != nil {
				in.AdditionalFields = tt.fields
			}

			got, err := NewRoleMetadataDeriver().DeriveRoleMetadata(t.Context(), 1, in, tt.action, failingResolver)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, source.MCMAddress, got.MCMAddress)
			require.Equal(t, source.StartingOpCount, got.StartingOpCount)

			want, err := SuiMetadata(source)
			require.NoError(t, err)
			want.Role = tt.wantRole
			derived, err := SuiMetadata(got)
			require.NoError(t, err)
			require.Equal(t, want, derived)
		})
	}
}
