package mcms

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

type Encoder interface {
	HashOperation(opCount uint32, metadata types.ChainMetadata, operation types.ChainOperation) (common.Hash, error)
	HashMetadata(metadata types.ChainMetadata) (common.Hash, error)
}
