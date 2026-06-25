package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
)

func TestTimelockRoleHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role sdk.TimelockRole
		want string
	}{
		{name: "admin", role: sdk.TimelockRoleAdmin, want: "ADMIN_ROLE"},
		{name: "bypasser", role: sdk.TimelockRoleBypasser, want: "BYPASSER_ROLE"},
		{name: "canceller", role: sdk.TimelockRoleCanceller, want: "CANCELLER_ROLE"},
		{name: "executor", role: sdk.TimelockRoleExecutor, want: "EXECUTOR_ROLE"},
		{name: "proposer", role: sdk.TimelockRoleProposer, want: "PROPOSER_ROLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := TimelockRoleHash(tt.role)
			require.NoError(t, err)
			require.Equal(t, crypto.Keccak256Hash([]byte(tt.want)), got)
		})
	}
}

func TestTimelockRoleHashRejectsInvalid(t *testing.T) {
	t.Parallel()

	_, err := TimelockRoleHash(sdk.TimelockRole(99))
	require.ErrorContains(t, err, "invalid timelock role")
}
