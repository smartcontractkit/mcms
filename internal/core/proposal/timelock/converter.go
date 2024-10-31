package timelock

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperation(
		t BatchChainOperation,
		timelockAddress common.Address,
		minDelay string,
		operation TimelockOperationType,
		predecessor common.Hash,
	) (types.ChainOperation, common.Hash, error)
}
