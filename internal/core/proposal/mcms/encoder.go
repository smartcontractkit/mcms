package mcms

import "github.com/ethereum/go-ethereum/common"

type Encoder interface {
	HashOperation(opCount uint32, operation ChainOperation) (common.Hash, error)
	HashMetadata(metadata ChainMetadata) (common.Hash, error)
}
