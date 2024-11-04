package timelock

import "github.com/ethereum/go-ethereum/common"

type Inspector interface {
	GetProposers(address string) ([]common.Address, error)
	GetExecutors(address string) ([]common.Address, error)
	GetBypassers(address string) ([]common.Address, error)
	GetCancellers(address string) ([]common.Address, error)
	isOperation(address string, opId [32]byte) (bool, error)
	isOperationPending(address string, opId [32]byte) (bool, error)
	isOperationReady(address string, opId [32]byte) (bool, error)
	isOperationDone(address string, opId [32]byte) (bool, error)
}
