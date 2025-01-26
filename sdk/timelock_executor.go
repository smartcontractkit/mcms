package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockExecutor is an interface for executing scheduled timelock operations.
type TimelockExecutor interface {
	TimelockInspector
	Execute(ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash) (types.MinedTransaction, error)
}
