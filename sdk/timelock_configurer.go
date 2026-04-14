package sdk

import (
	"context"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConfigurer is an interface for configuring timelock contract parameters.
type TimelockConfigurer interface {
	TimelockInspector
	UpdateDelay(ctx context.Context, timelockAddress string, newDelay uint64) (types.TransactionResult, error)
}
