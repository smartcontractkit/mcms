package mcms

import (
	"errors"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm"
)

func NewEncoder(chainSelector mcms.ChainSelector, txCount uint64, overridePreviousRoot bool, isSim bool) (mcms.Encoder, error) {
	chain, exists := chain_selectors.ChainBySelector(uint64(chainSelector))
	if !exists {
		return nil, &core.InvalidChainIDError{
			ReceivedChainID: uint64(chainSelector),
		}
	}

	// Simulated chains always have block.chainid = 1337
	// So for setRoot to execute (not throw WrongChainId) we must
	// override the evmChainID to be 1337.
	if isSim {
		chain.EvmChainID = 1337
	}

	family, err := chain_selectors.GetSelectorFamily(uint64(chainSelector))
	if err != nil {
		return nil, errors.New("unknown chain family")
	}

	var encoder mcms.Encoder
	switch family {
	case chain_selectors.FamilyEVM:
		encoder = evm.NewEVMEncoder(
			txCount,
			chain.EvmChainID,
			overridePreviousRoot,
		)
	case chain_selectors.FamilySolana:
		return nil, errors.New("solana not supported")
	case chain_selectors.FamilyStarknet:
		return nil, errors.New("starknet not supported")
	default:
		return nil, errors.New("unsupported chain family")
	}

	return encoder, nil
}
