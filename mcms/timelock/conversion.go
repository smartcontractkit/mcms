package timelock

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms"
	"github.com/smartcontractkit/mcms/pkg/proposal/timelock"
)

type TimelockConverter interface {
	ConvertBatchToChainOperation(
		t timelock.BatchChainOperation,
		timelockAddress common.Address,
		minDelay string,
		operation timelock.TimelockOperation,
		predecessor common.Hash,
	) (mcms.ChainOperation, common.Hash, error)
}
