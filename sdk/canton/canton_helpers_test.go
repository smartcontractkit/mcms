package canton

import (
	"testing"

	"github.com/stretchr/testify/require"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

func TestCantonRoleFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		action       mcmstypes.TimelockAction
		expectedRole TimelockRole
		expectError  bool
	}{
		{
			name:         "bypass action returns bypasser role",
			action:       mcmstypes.TimelockActionBypass,
			expectedRole: TimelockRoleBypasser,
		},
		{
			name:         "schedule action returns proposer role",
			action:       mcmstypes.TimelockActionSchedule,
			expectedRole: TimelockRoleProposer,
		},
		{
			name:         "cancel action returns canceller role",
			action:       mcmstypes.TimelockActionCancel,
			expectedRole: TimelockRoleCanceller,
		},
		{
			name:        "unknown action returns error",
			action:      mcmstypes.TimelockAction("unknown"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := CantonRoleFromAction(tt.action)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedRole, role)
		})
	}
}
