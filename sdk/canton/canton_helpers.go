package canton

import (
	"errors"

	"github.com/smartcontractkit/mcms/types"
)

func CantonRoleFromAction(action types.TimelockAction) (TimelockRole, error) {
	switch action {
	case types.TimelockActionBypass:
		return TimelockRoleBypasser, nil
	case types.TimelockActionSchedule:
		return TimelockRoleProposer, nil
	case types.TimelockActionCancel:
		return TimelockRoleCanceller, nil
	default:
		return 0, errors.New("unknown timelock action")
	}
}
