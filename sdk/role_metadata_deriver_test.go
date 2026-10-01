package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockRoleFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		action  types.TimelockAction
		want    TimelockRole
		wantErr string
	}{
		{action: types.TimelockActionSchedule, want: TimelockRoleProposer},
		{action: types.TimelockActionCancel, want: TimelockRoleCanceller},
		{action: types.TimelockActionBypass, want: TimelockRoleBypasser},
		{action: types.TimelockAction("unknown"), wantErr: "unknown timelock action"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			t.Parallel()

			got, err := TimelockRoleFromAction(tt.action)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestResolveRoleMetadata(t *testing.T) {
	t.Parallel()

	const selector = types.ChainSelector(1)
	source := types.ChainMetadata{
		StartingOpCount:  3,
		MCMAddress:       "0xproposer",
		AdditionalFields: json.RawMessage(`{"k":"v"}`),
	}

	tests := []struct {
		name     string
		action   types.TimelockAction
		resolver RoleAddressResolver
		wantAddr string
		wantErr  string
	}{
		{
			name:   "cancel resolves the canceller address",
			action: types.TimelockActionCancel,
			resolver: RoleAddressResolverFunc(func(_ context.Context, req RoleAddressRequest) (string, error) {
				require.Equal(t, RoleAddressRequest{
					Selector:         selector,
					Role:             TimelockRoleCanceller,
					SourceMCMAddress: "0xproposer",
				}, req)

				return "0xcanceller", nil
			}),
			wantAddr: "0xcanceller",
		},
		{
			name:   "bypass resolves the bypasser address",
			action: types.TimelockActionBypass,
			resolver: RoleAddressResolverFunc(func(_ context.Context, req RoleAddressRequest) (string, error) {
				require.Equal(t, TimelockRoleBypasser, req.Role)

				return "0xbypasser", nil
			}),
			wantAddr: "0xbypasser",
		},
		{
			name:    "nil resolver",
			action:  types.TimelockActionCancel,
			wantErr: "role address resolver is required",
		},
		{
			name:   "resolver error",
			action: types.TimelockActionCancel,
			resolver: RoleAddressResolverFunc(func(context.Context, RoleAddressRequest) (string, error) {
				return "", errors.New("boom")
			}),
			wantErr: "failed to resolve Canceller MCM address for chain 1: boom",
		},
		{
			name:   "empty address",
			action: types.TimelockActionCancel,
			resolver: RoleAddressResolverFunc(func(context.Context, RoleAddressRequest) (string, error) {
				return "", nil
			}),
			wantErr: "empty Canceller MCM address",
		},
		{
			name:    "unknown action",
			action:  types.TimelockAction("unknown"),
			wantErr: "unknown timelock action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveRoleMetadata(t.Context(), selector, source, tt.action, tt.resolver)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantAddr, got.MCMAddress)
			require.Equal(t, source.StartingOpCount, got.StartingOpCount)
			require.JSONEq(t, string(source.AdditionalFields), string(got.AdditionalFields))
			require.Equal(t, "0xproposer", source.MCMAddress, "source must not be mutated")
		})
	}
}
