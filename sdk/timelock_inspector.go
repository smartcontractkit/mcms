package sdk

import (
	"context"
)

type TimelockInspector[T any] interface {
	GetProposers(ctx context.Context, address string) ([]T, error)
	GetExecutors(ctx context.Context, address string) ([]T, error)
	GetBypassers(ctx context.Context, address string) ([]T, error)
	GetCancellers(ctx context.Context, address string) ([]T, error)
	IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error)
	IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error)
}
