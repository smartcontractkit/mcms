package mcms

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

type Simulator interface {
	SimulateSetRoot(
		metadata types.ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []types.Signature,
	) (bool, error)
	SimulateOperation(nonce uint32, proof []common.Hash, operation types.ChainOperation) (bool, error)
}
