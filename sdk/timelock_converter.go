package sdk

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperation(
		bop types.BatchOperation,
		timelockAddress string,
		delay types.Duration,
		action types.TimelockAction,
		predecessor common.Hash,
	) (types.Operation, common.Hash, error)
}
