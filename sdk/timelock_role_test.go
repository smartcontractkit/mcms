package sdk

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestTimelockRole_Hash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role TimelockRole
		want string
	}{
		{name: "admin", role: TimelockRoleAdmin, want: "ADMIN_ROLE"},
		{name: "bypasser", role: TimelockRoleBypasser, want: "BYPASSER_ROLE"},
		{name: "canceller", role: TimelockRoleCanceller, want: "CANCELLER_ROLE"},
		{name: "executor", role: TimelockRoleExecutor, want: "EXECUTOR_ROLE"},
		{name: "proposer", role: TimelockRoleProposer, want: "PROPOSER_ROLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.role.Hash()
			require.NoError(t, err)
			require.Equal(t, crypto.Keccak256Hash([]byte(tt.want)), got)
		})
	}
}

func TestTimelockRole_Valid(t *testing.T) {
	t.Parallel()

	require.True(t, TimelockRoleAdmin.Valid())
	require.False(t, TimelockRole(99).Valid())
}

func TestTimelockRole_HashRejectsInvalid(t *testing.T) {
	t.Parallel()

	_, err := TimelockRole(99).Hash()
	require.ErrorContains(t, err, "invalid timelock role")
}
