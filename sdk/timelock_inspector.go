package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

type TimelockInspector interface {
	GetProposers(ctx context.Context, timelockID types.ContractID) ([]common.Address, error)
	GetExecutors(ctx context.Context, timelockID types.ContractID) ([]common.Address, error)
	GetBypassers(ctx context.Context, timelockID types.ContractID) ([]common.Address, error)
	GetCancellers(ctx context.Context, timelockID types.ContractID) ([]common.Address, error)
	IsOperation(ctx context.Context, timelockID types.ContractID, opID [32]byte) (bool, error)
	IsOperationPending(ctx context.Context, timelockID types.ContractID, opID [32]byte) (bool, error)
	IsOperationReady(ctx context.Context, timelockID types.ContractID, opID [32]byte) (bool, error)
	IsOperationDone(ctx context.Context, timelockID types.ContractID, opID [32]byte) (bool, error)
}
