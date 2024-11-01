package timelock

import "github.com/ethereum/go-ethereum/common"

type Inspector interface {
	GetProposers(timelockAddress string) ([]common.Address, error)
	GetExecutors(timelockAddress string) ([]common.Address, error)
	GetBypassers(timelockAddress string) ([]common.Address, error)
	GetCancellers(timelockAddress string) ([]common.Address, error)
	isOperation(timelockAddress string)
	isOperationPending(timelockAddress string)
	isOperationReady(timelockAddress string)
	isOperationDone(timelockAddress string)
}
