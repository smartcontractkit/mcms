package sdk

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/config"
	"github.com/smartcontractkit/mcms/types"
)

type Inspector interface {
	GetConfig(mcmAddress string) (*config.Config, error)
	GetOpCount(mcmAddress string) (uint64, error)
	GetRoot(mcmAddress string) (common.Hash, uint32, error)
	GetRootMetadata(mcmAddress string) (types.ChainMetadata, error)
}

type Encoder interface {
	HashOperation(opCount uint32, metadata types.ChainMetadata, operation types.ChainOperation) (common.Hash, error)
	HashMetadata(metadata types.ChainMetadata) (common.Hash, error)
}
