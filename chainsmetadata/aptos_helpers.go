package chainsmetadata

import (
	"errors"

	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func AptosRoleFromAction(action types.TimelockAction) (aptos.TimelockRole, error) {
	switch action {
	case types.TimelockActionBypass:
		return aptos.TimelockRoleBypasser, nil
	case types.TimelockActionSchedule:
		return aptos.TimelockRoleProposer, nil
	case types.TimelockActionCancel:
		return aptos.TimelockRoleCanceller, nil
	default:
		return 0, errors.New("unknown timelock action")
	}
}
