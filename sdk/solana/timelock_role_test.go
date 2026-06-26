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
		name string
		role sdk.TimelockRole
		want bindings.Role
	}{
		{name: "proposer", role: sdk.TimelockRoleProposer, want: bindings.Proposer_Role},
		{name: "executor", role: sdk.TimelockRoleExecutor, want: bindings.Executor_Role},
		{name: "canceller", role: sdk.TimelockRoleCanceller, want: bindings.Canceller_Role},
		{name: "bypasser", role: sdk.TimelockRoleBypasser, want: bindings.Bypasser_Role},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := TimelockRoleToBinding(tt.role)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTimelockRoleToBindingRejectsAdmin(t *testing.T) {
	t.Parallel()

	_, err := TimelockRoleToBinding(sdk.TimelockRoleAdmin)
	require.EqualError(t, err, "admin role is not grantable via access controller on solana")
}

func TestTimelockRoleToBindingRejectsInvalid(t *testing.T) {
	t.Parallel()

	_, err := TimelockRoleToBinding(sdk.TimelockRole(99))
	require.EqualError(t, err, "invalid timelock role: 99")
}
