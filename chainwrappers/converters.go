package chainwrappers

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// BuildConverters constructs a map of chain selectors to their respective timelock converters based on the provided timelock proposal.
func BuildConverters(chainMetadata map[types.ChainSelector]types.ChainMetadata) (map[types.ChainSelector]sdk.TimelockConverter, error) {
	converters := make(map[types.ChainSelector]sdk.TimelockConverter)
	for chainMeta := range chainMetadata {
		fam, err := types.GetChainSelectorFamily(chainMeta)
		if err != nil {
			return nil, fmt.Errorf("error getting chain family: %w", err)
		}

		var converter sdk.TimelockConverter
		switch fam {
		case chainsel.FamilyEVM:
			converter = evm.NewTimelockConverter()
		case chainsel.FamilySolana:
			converter = solana.NewTimelockConverter()
		case chainsel.FamilyAptos:
			converter = aptos.NewTimelockConverter()
		case chainsel.FamilySui:
			converter, _ = sui.NewTimelockConverter()
		case chainsel.FamilyTon:
			converter = ton.NewTimelockConverter(ton.DefaultSendAmount)
		default:
			return nil, fmt.Errorf("unsupported chain family %s", fam)
		}

		converters[chainMeta] = converter
	}

	return converters, nil
}
