package mcms

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/canton"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"

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
	case chainsel.FamilyEVM:
		encoder = evm.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
			isSim,
		)
	case chainsel.FamilySolana:
		encoder = solana.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
			// isSim,
		)
	case chainsel.FamilyAptos:
		encoder = aptos.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
		)
	case chainsel.FamilySui:
		encoder = sui.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
		)
	case chainsel.FamilyTon:
		encoder = ton.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
		)
	case chainsel.FamilyCanton:
		encoder = canton.NewEncoder(
			csel,
			txCount,
			overridePreviousRoot,
		)
	}

	return encoder, nil
}

// newTimelockConverter a new TimelockConverter that can convert timelock proposals
// for the given chain. The metadata parameter is used to select the correct
// converter variant (e.g. curse_mcms on Aptos).
func newTimelockConverter(csel types.ChainSelector, metadata types.ChainMetadata) (sdk.TimelockConverter, error) {
	return chainwrappers.BuildConverter(csel, metadata)
}

func operationIDFn(_ context.Context, csel types.ChainSelector) (sdk.OperationID, error) {
	family, err := types.GetChainSelectorFamily(csel) //nolint:contextcheck //OPT-400
	if err != nil {
		return nil, err
	}

	switch family {
	case chainsel.FamilyEVM:
		return evm.OperationID, nil
	case chainsel.FamilySolana:
		return solana.OperationID, nil
	case chainsel.FamilyAptos:
		return aptos.OperationID, nil
	case chainsel.FamilySui:
		return sui.OperationID, nil
	case chainsel.FamilyTon:
		return ton.OperationID, nil
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}
