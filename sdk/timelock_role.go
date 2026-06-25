package sdk

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TimelockRole identifies one of the supported RBACTimelock roles.
type TimelockRole uint8

const (
	TimelockRoleAdmin TimelockRole = iota
	TimelockRoleBypasser
	TimelockRoleCanceller
	TimelockRoleExecutor
	TimelockRoleProposer
)

type timelockRoleInfo struct {
	name     string
	hashName string
}

var timelockRoles = map[TimelockRole]timelockRoleInfo{
	TimelockRoleAdmin:     {name: "Admin", hashName: "ADMIN_ROLE"},
	TimelockRoleBypasser:  {name: "Bypasser", hashName: "BYPASSER_ROLE"},
	TimelockRoleCanceller: {name: "Canceller", hashName: "CANCELLER_ROLE"},
	TimelockRoleExecutor:  {name: "Executor", hashName: "EXECUTOR_ROLE"},
	TimelockRoleProposer:  {name: "Proposer", hashName: "PROPOSER_ROLE"},
}

// Valid reports whether r is one of the supported timelock roles.
func (r TimelockRole) Valid() bool {
	_, ok := timelockRoles[r]
	return ok
}

// Hash returns the EVM bytes32 representation of the timelock role.
func (r TimelockRole) Hash() (common.Hash, error) {
	info, ok := timelockRoles[r]
	if !ok {
		return common.Hash{}, fmt.Errorf("invalid timelock role: %d", r)
	}

	return crypto.Keccak256Hash([]byte(info.hashName)), nil
}

func (r TimelockRole) String() string {
	if info, ok := timelockRoles[r]; ok {
		return info.name
	}
	return "Unknown"
}
