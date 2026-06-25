package evm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/sdk"
)

var timelockRoleHashNames = map[sdk.TimelockRole]string{
	sdk.TimelockRoleAdmin:     "ADMIN_ROLE",
	sdk.TimelockRoleBypasser:  "BYPASSER_ROLE",
	sdk.TimelockRoleCanceller: "CANCELLER_ROLE",
	sdk.TimelockRoleExecutor:  "EXECUTOR_ROLE",
	sdk.TimelockRoleProposer:  "PROPOSER_ROLE",
}

// TimelockRoleHash returns the RBACTimelock AccessControl role hash for role.
func TimelockRoleHash(role sdk.TimelockRole) (common.Hash, error) {
	hashName, ok := timelockRoleHashNames[role]
	if !ok {
		return common.Hash{}, fmt.Errorf("invalid timelock role: %d", role)
	}

	return crypto.Keccak256Hash([]byte(hashName)), nil
}
