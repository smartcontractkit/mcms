package mcms

import (
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func NewEncoder(chainSelector types.ChainSelector, txCount uint64, overridePreviousRoot bool, isSim bool) (sdk.Encoder, error) {
	chain, exists := cselectors.ChainBySelector(uint64(chainSelector))
	if !exists {
		return nil, &core.InvalidChainIDError{
			ReceivedChainID: uint64(chainSelector),
		}
	}

	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, err
	}

	var encoder sdk.Encoder
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
