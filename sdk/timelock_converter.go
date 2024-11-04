package sdk

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperation(
		t types.BatchChainOperation,
		timelockAddress string,
		minDelay string,
		operation types.TimelockAction,
		predecessor common.Hash,
	) (types.ChainOperation, common.Hash, error)
}
