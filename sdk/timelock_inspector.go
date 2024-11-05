package sdk

import (
	"github.com/ethereum/go-ethereum/common"
)

type TimelockInspector interface {
	GetProposers(address string) ([]common.Address, error)
	GetExecutors(address string) ([]common.Address, error)
	GetBypassers(address string) ([]common.Address, error)
	GetCancellers(address string) ([]common.Address, error)
	IsOperation(address string, opID [32]byte) (bool, error)
	IsOperationPending(address string, opID [32]byte) (bool, error)
	IsOperationReady(address string, opID [32]byte) (bool, error)
	IsOperationDone(address string, opID [32]byte) (bool, error)
}
