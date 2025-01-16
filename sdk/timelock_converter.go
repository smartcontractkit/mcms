package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperations(
		ctx context.Context,
		bop types.BatchOperation,
		timelockAddress string,
		mcmAddress string,
		delay types.Duration,
		action types.TimelockAction,
		predecessor common.Hash,
		salt common.Hash,
	) ([]types.Operation, common.Hash, error)
}
