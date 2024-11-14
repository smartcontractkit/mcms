package mcms

import (
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// newEncoder returns a new Encoder that can encode operations and metadata for the given chain.
// Additional arguments are used to configure the encoder.
func newEncoder(
	csel types.ChainSelector, txCount uint64, overridePreviousRoot bool, isSim bool,
) (sdk.Encoder, error) {
	chain, exists := cselectors.ChainBySelector(uint64(csel))
	if !exists {
		return nil, &sdkerrors.InvalidChainIDError{
			ReceivedChainID: csel,
		}
	}

	family, err := types.GetChainSelectorFamily(csel)
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

		encoder = evm.NewEncoder(
			txCount,
			chain.EvmChainID,
			overridePreviousRoot,
		)
	}

	return encoder, nil
}
