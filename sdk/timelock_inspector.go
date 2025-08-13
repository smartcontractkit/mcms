package sdk

import (
	"context"
)

type TimelockInspector interface {
	GetProposers(ctx context.Context, address string) ([]string, error)
	GetExecutors(ctx context.Context, address string) ([]string, error)
	GetBypassers(ctx context.Context, address string) ([]string, error)
	GetCancellers(ctx context.Context, address string) ([]string, error)
	IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error)
	GetMinDelay(ctx context.Context, address string) (uint64, error)
}
