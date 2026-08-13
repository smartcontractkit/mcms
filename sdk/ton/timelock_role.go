package ton

import (
	"fmt"
	"math/big"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"

	"github.com/smartcontractkit/mcms/sdk"
)

var timelockRoleHashes = map[sdk.TimelockRole]*big.Int{
	sdk.TimelockRoleAdmin:     timelock.RoleAdmin,
	sdk.TimelockRoleBypasser:  timelock.RoleBypasser,
	sdk.TimelockRoleCanceller: timelock.RoleCanceller,
	sdk.TimelockRoleExecutor:  timelock.RoleExecutor,
	sdk.TimelockRoleProposer:  timelock.RoleProposer,
}

// TimelockRoleHash returns the RBACTimelock AccessControl role hash for role.
func TimelockRoleHash(role sdk.TimelockRole) (*big.Int, error) {
	hash, ok := timelockRoleHashes[role]
	if !ok {
		return nil, fmt.Errorf("invalid timelock role: %d", role)
	}

	return new(big.Int).Set(hash), nil
}
