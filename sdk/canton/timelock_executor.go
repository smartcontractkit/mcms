package canton

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is a stub implementation for Canton timelock execution.
// It embeds TimelockInspector; Execute returns "not implemented" until Phase C.
type TimelockExecutor struct {
	*TimelockInspector
}

// NewTimelockExecutor creates a new TimelockExecutor (stub).
func NewTimelockExecutor() *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(),
	}
}

func (t *TimelockExecutor) Execute(
	ctx context.Context,
	bop types.BatchOperation,
	timelockAddress string,
	predecessor common.Hash,
	salt common.Hash,
) (types.TransactionResult, error) {
	return types.TransactionResult{}, errors.New("not implemented on Canton")
}
