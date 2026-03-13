package aptos

import (
	"errors"

	"github.com/smartcontractkit/mcms/types"
)

func AptosRoleFromAction(action types.TimelockAction) (TimelockRole, error) {
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

func ActionFromAptosRole(role TimelockRole) (types.TimelockAction, error) {
	switch role {
	case TimelockRoleBypasser:
		return types.TimelockActionBypass, nil
	case TimelockRoleProposer:
		return types.TimelockActionSchedule, nil
	case TimelockRoleCanceller:
		return types.TimelockActionCancel, nil
	default:
		return "unknown", errors.New("unknown timelock role")
	}
}
