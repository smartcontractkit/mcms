package utils

import (
	"errors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func AptosRoleFromProposal(proposal *mcms.TimelockProposal) (aptos.TimelockRole, error) {
	if proposal == nil {
		return 0, errors.New("aptos timelock proposal is needed")
	}

	role, err := AptosRoleFromAction(proposal.Action)
	if err != nil {
		return 0, err
	}

	return role, nil
}

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
