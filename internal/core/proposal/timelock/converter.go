package timelock

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperation(
		t BatchChainOperation,
		timelockAddress common.Address,
		minDelay string,
		operation TimelockOperationType,
		predecessor common.Hash,
	) (mcms.ChainOperation, common.Hash, error)
}
