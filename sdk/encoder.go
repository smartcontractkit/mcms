package sdk

import (
	"github.com/ethereum/go-ethereum/common"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// Encoder encoding MCMS operations and metadata into hashes.
type Encoder interface {
	HashOperation(opCount uint32, metadata types.ChainMetadata, op types.ChainOperation) (common.Hash, error)
	HashMetadata(metadata types.ChainMetadata) (common.Hash, error)
}

// NewEncoder returns a new Encoder that can encode operations and metadata for the given chain.
// Additional arguments are used to configure the encoder.
func NewEncoder(
	csel types.ChainSelector, txCount uint64, overridePreviousRoot bool, isSim bool,
) (Encoder, error) {
	chain, exists := cselectors.ChainBySelector(uint64(csel))
	if !exists {
		return nil, &core.InvalidChainIDError{
			ReceivedChainID: uint64(csel),
		}
	}

	family, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return nil, err
	}

	var encoder Encoder
	switch family {
	case cselectors.FamilyEVM:
		// Simulated chains always have block.chainid = 1337
		// So for setRoot to execute (not throw WrongChainId) we must
		// override the evmChainID to be 1337.
		if isSim {
			chain.EvmChainID = 1337
		}

		encoder = evm.NewEVMEncoder(
			txCount,
			chain.EvmChainID,
			overridePreviousRoot,
		)
	}

	return encoder, nil
}
