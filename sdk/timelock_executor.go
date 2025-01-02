package sdk

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/types"
)

// TimelockExecutor is an interface for executing scheduled timelock operations.
type TimelockExecutor interface {
	Execute(bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash) (string, error)
}
