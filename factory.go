package mcms

import (
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// newEncoder returns a new Encoder that can encode operations and metadata for the given chain.
// Additional arguments are used to configure the encoder.
func newEncoder(
	csel types.ChainSelector, txCount uint64, overridePreviousRoot bool, isSim bool,
) (sdk.Encoder, error) {
	family, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return nil, err
	}

	var encoder sdk.Encoder
	switch family {
	case cselectors.FamilyEVM:
		encoder = evm.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
			isSim,
		)
	}

	return encoder, nil
}
