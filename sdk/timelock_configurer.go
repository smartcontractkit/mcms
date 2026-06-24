package sdk

import (
	"context"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConfigurer is an interface for configuring timelock contract parameters.
type TimelockConfigurer interface {
	UpdateDelay(ctx context.Context, timelockAddress string, newDelay uint64) (types.TransactionResult, error)
	GrantRoles(
		ctx context.Context,
		timelockAddress string,
		role TimelockRole,
		addresses []string,
	) (types.TransactionResult, error)
}
