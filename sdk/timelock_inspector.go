package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type TimelockInspector interface {
	GetProposers(ctx context.Context, address string) ([]common.Address, error)
	GetExecutors(ctx context.Context, address string) ([]common.Address, error)
	GetBypassers(ctx context.Context, address string) ([]common.Address, error)
	GetCancellers(ctx context.Context, address string) ([]common.Address, error)
	IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error)
}
