package sdk

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// Encoder encoding MCMS operations and metadata into hashes.
type Encoder interface {
	HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error)
	HashMetadata(metadata types.ChainMetadata) (common.Hash, error)
}
