package sdk

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// Inspector is an interface for inspecting on chain state of MCMS contracts.
type Inspector interface {
	GetConfig(ctx context.Context, mcmAddr string) (*types.Config, error)
	GetOpCount(ctx context.Context, mcmAddr string) (uint64, error)
	GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error)
	GetRootMetadata(ctx context.Context, mcmAddr string) (types.ChainMetadata, error)
}
