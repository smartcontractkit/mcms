package stellar

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	timelockbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/timelock"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor submits execute_batch on Soroban RBACTimelock via [bindings.Invoker].
type TimelockExecutor struct {
	*TimelockInspector
	invoker        bindings.Invoker
	executorCaller string
}

// NewTimelockExecutor builds an executor that invokes execute_batch as executorCaller (must hold
// EXECUTOR on the timelock contract).
func NewTimelockExecutor(invoker bindings.Invoker, executorCaller string) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(invoker),
		invoker:           invoker,
		executorCaller:    executorCaller,
	}
}

// Execute invokes execute_batch with the batch calls, predecessor, and salt (operation id must match).
func (e *TimelockExecutor) Execute(
	ctx context.Context,
	bop types.BatchOperation,
	timelockAddress string,
	predecessor common.Hash,
	salt common.Hash,
) (types.TransactionResult, error) {
	if e.executorCaller == "" {
		return types.TransactionResult{}, fmt.Errorf("stellar timelock: executor caller address is empty")
	}

	id, err := normalizeContractIDStrkey(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}

	calls, err := callsFromBatchOperation(bop)
	if err != nil {
		return types.TransactionResult{}, err
	}

	client := timelockbindings.NewTimelockClient(e.invoker, id)

	tbCalls := timelockbindings.Calls{Inner: calls}
	if err := client.ExecuteBatch(ctx, e.executorCaller, tbCalls, predecessor, salt); err != nil {
		return types.TransactionResult{}, err
	}

	return stellarTransactionResult(e.invoker), nil
}
