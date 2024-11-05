package sdk

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperation(
		batchOps types.BatchChainOperation,
		timelockAddress string,
		delay string,
		action types.TimelockAction,
		predecessor common.Hash,
	) (types.ChainOperation, common.Hash, error)
}
