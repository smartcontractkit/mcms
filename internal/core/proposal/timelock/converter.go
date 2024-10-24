package timelock

import (
	"time"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

type TimelockConverter interface {
	// Converts a timelock 'BatchChainOperation' to a timelock wrapped tx as a MCMS 'ChainOperation'
	Convert(
		op BatchChainOperation,
		timelockAddress string,
		delay time.Duration,
		operation TimelockOperation,
	) (mcms.ChainOperation, error)
}
