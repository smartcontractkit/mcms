package solana

import (
	"testing"

	"github.com/stretchr/testify/require"

	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"

	"github.com/smartcontractkit/mcms/sdk"
)

func TestTimelockRoleToBinding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    sdk.TimelockRole
		want    bindings.Role
		wantErr string
	}{
		{name: "proposer", role: sdk.TimelockRoleProposer, want: bindings.Proposer_Role},
		{name: "executor", role: sdk.TimelockRoleExecutor, want: bindings.Executor_Role},
		{name: "canceller", role: sdk.TimelockRoleCanceller, want: bindings.Canceller_Role},
		{name: "bypasser", role: sdk.TimelockRoleBypasser, want: bindings.Bypasser_Role},
		{name: "admin", role: sdk.TimelockRoleAdmin, wantErr: "admin role is not grantable via access controller on solana"},
		{name: "invalid", role: sdk.TimelockRole(99), wantErr: "invalid timelock role: 99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := TimelockRoleToBinding(tt.role)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
