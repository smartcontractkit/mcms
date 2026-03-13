package aptos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestAptosRoleFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		action       mcmsTypes.TimelockAction
		expectedRole TimelockRole
		expectError  bool
	}{
		{
			name:         "bypass action returns bypasser role",
			action:       mcmsTypes.TimelockActionBypass,
			expectedRole: TimelockRoleBypasser,
			expectError:  false,
		},
		{
			name:         "schedule action returns proposer role",
			action:       mcmsTypes.TimelockActionSchedule,
			expectedRole: TimelockRoleProposer,
			expectError:  false,
		},
		{
			name:         "cancel action returns canceller role",
			action:       mcmsTypes.TimelockActionCancel,
			expectedRole: TimelockRoleCanceller,
			expectError:  false,
		},
		{
			name:         "unknown action returns error",
			action:       mcmsTypes.TimelockAction("unknown"),
			expectedRole: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := AptosRoleFromAction(tt.action)

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, "unknown timelock action", err.Error())
				assert.Equal(t, tt.expectedRole, role)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
		})
	}
}

func TestActionFromAptosRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    TimelockRole
		want    mcmsTypes.TimelockAction
		wantErr string
	}{
		{
			name: "bypasser role returns bypass action",
			role: TimelockRoleBypasser,
			want: mcmsTypes.TimelockActionBypass,
		},
		{
			name: "proposer role returns schedule action",
			role: TimelockRoleProposer,
			want: mcmsTypes.TimelockActionSchedule,
		},
		{
			name: "canceller role returns cancel action",
			role: TimelockRoleCanceller,
			want: mcmsTypes.TimelockActionCancel,
		},
		{
			name:    "unknown role returns error",
			role:    TimelockRole(99),
			want:    "unknown",
			wantErr: "unknown timelock role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ActionFromAptosRole(tt.role)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
