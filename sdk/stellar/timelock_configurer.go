package stellar

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

// TimelockConfigurer updates timelock parameters on Soroban RBACTimelock.
type TimelockConfigurer struct {
	TimelockInspector
	invoker     bindings.Invoker
	adminCaller string
}

// NewTimelockConfigurer returns a configurer that invokes update_delay as adminCaller (must hold
// ADMIN on the timelock contract).
func NewTimelockConfigurer(invoker bindings.Invoker, adminCaller string) *TimelockConfigurer {
	return &TimelockConfigurer{
		TimelockInspector: *NewTimelockInspector(invoker),
		invoker:           invoker,
		adminCaller:       adminCaller,
	}
}

// UpdateDelay calls update_delay on the timelock contract.
func (c *TimelockConfigurer) UpdateDelay(
	ctx context.Context, timelockAddress string, newDelay uint64,
) (types.TransactionResult, error) {
	if c.adminCaller == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar timelock: admin caller address is empty")
	}

	id, err := normalizeContractIDStrkey(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}

	client := timelockbindings.NewTimelockClient(c.invoker, id)
	if err := client.UpdateDelay(ctx, c.adminCaller, newDelay); err != nil {
		return types.TransactionResult{}, err
	}

	return stellarTransactionResult(c.invoker), nil
}
