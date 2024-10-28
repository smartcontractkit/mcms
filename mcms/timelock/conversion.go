package timelock

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
	timelockTypes "github.com/smartcontractkit/mcms/pkg/proposal/timelock/types"
)

type TimelockConverter interface {
	ConvertBatchToChainOperation(
		t timelockTypes.BatchChainOperation,
		timelockAddress common.Address,
		minDelay string,
		operation timelockTypes.TimelockOperationType,
		predecessor common.Hash,
	) (types.ChainOperation, common.Hash, error)
}
