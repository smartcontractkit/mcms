package sdk

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TimelockRole identifies one of the supported RBACTimelock roles.
type TimelockRole uint8

const (
	TimelockRoleUnknown TimelockRole = iota
	TimelockRoleAdmin
	TimelockRoleBypasser
	TimelockRoleCanceller
	TimelockRoleExecutor
	TimelockRoleProposer
)

// Hash returns the EVM bytes32 representation of the timelock role.
func (r TimelockRole) Hash() (common.Hash, error) {
	switch r {
	case TimelockRoleAdmin:
		return crypto.Keccak256Hash([]byte("ADMIN_ROLE")), nil
	case TimelockRoleBypasser:
		return crypto.Keccak256Hash([]byte("BYPASSER_ROLE")), nil
	case TimelockRoleCanceller:
		return crypto.Keccak256Hash([]byte("CANCELLER_ROLE")), nil
	case TimelockRoleExecutor:
		return crypto.Keccak256Hash([]byte("EXECUTOR_ROLE")), nil
	case TimelockRoleProposer:
		return crypto.Keccak256Hash([]byte("PROPOSER_ROLE")), nil
	default:
		return common.Hash{}, fmt.Errorf("invalid timelock role: %d", r)
	}
}

func (r TimelockRole) String() string {
	switch r {
	case TimelockRoleAdmin:
		return "Admin"
	case TimelockRoleBypasser:
		return "Bypasser"
	case TimelockRoleCanceller:
		return "Canceller"
	case TimelockRoleExecutor:
		return "Executor"
	case TimelockRoleProposer:
		return "Proposer"
	default:
		return "Unknown"
	}
}
