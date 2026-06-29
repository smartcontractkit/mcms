package solana

import (
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"

	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
)

var timelockRoleBindings = map[sdk.TimelockRole]bindings.Role{
	sdk.TimelockRoleProposer:  bindings.Proposer_Role,
	sdk.TimelockRoleExecutor:  bindings.Executor_Role,
	sdk.TimelockRoleCanceller: bindings.Canceller_Role,
	sdk.TimelockRoleBypasser:  bindings.Bypasser_Role,
}

// TimelockRoleToBinding maps sdk.TimelockRole to the Solana timelock program Role enum.
func TimelockRoleToBinding(role sdk.TimelockRole) (bindings.Role, error) {
	if role == sdk.TimelockRoleAdmin {
		return bindings.Role(0), fmt.Errorf("admin role is not grantable via access controller on solana")
	}

	bindingRole, ok := timelockRoleBindings[role]
	if !ok {
		return bindings.Role(0), fmt.Errorf("invalid timelock role: %d", role)
	}

	return bindingRole, nil
}
