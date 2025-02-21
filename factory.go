package mcms

import (
	"fmt"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
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
	case cselectors.FamilySolana:
		encoder = solana.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
			// isSim,
		)
	}

	return encoder, nil
}

// newTimelockConverterFromExecutor returns a new TimelockConverter that can convert timelock proposals
// for the given chain.
func newTimelockConverterFromExecutor(
	csel types.ChainSelector,
	executor sdk.TimelockExecutor,
) (sdk.TimelockConverter, error) {
	family, err := types.GetChainSelectorFamily(csel)
	if err != nil {
		return nil, err
	}

	switch family {
	case cselectors.FamilyEVM:
		return &evm.TimelockConverter{}, nil

	case cselectors.FamilySolana:
		return &solana.TimelockConverter{}, nil

	default:
		return nil, fmt.Errorf("unsupported executor type: %T", executor)
	}
}
