package mcms

import "github.com/ethereum/go-ethereum/common"

type Simulator interface {
	SimulateSetRoot(
		metadata ChainMetadata,
		proof []common.Hash,
		root [32]byte,
		validUntil uint32,
		sortedSignatures []Signature,
	) (bool, error)
	SimulateOperation(nonce uint32, proof []common.Hash, operation ChainOperation) (bool, error)
}
