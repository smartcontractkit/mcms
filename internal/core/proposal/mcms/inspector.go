package mcms

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/config"
)

type Inspector interface {
	GetConfig(mcmAddress string) (*config.Config, error)
	GetOpCount(mcmAddress string) (uint64, error)
	GetRoot(mcmAddress string) (common.Hash, uint32, error)
	GetRootMetadata(mcmAddress string) (ChainMetadata, error)
}
