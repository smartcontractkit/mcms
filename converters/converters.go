package proposalutils

import (
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

// BuildConvertersForTimelockProposal constructs a map of chain selectors to their respective timelock converters based on the provided timelock proposal.
func BuildConvertersForTimelockProposal(proposal mcms.TimelockProposal) (map[types.ChainSelector]sdk.TimelockConverter, error) {
	converters := make(map[types.ChainSelector]sdk.TimelockConverter)
	for chainMeta := range proposal.ChainMetadata {
		fam, err := types.GetChainSelectorFamily(chainMeta)
		if err != nil {
			return nil, fmt.Errorf("error getting chain family: %w", err)
		}

		var converter sdk.TimelockConverter
		switch fam {
		case chainsel.FamilyEVM:
			converter = evm.TimelockConverter{}
		case chainsel.FamilySolana:
			converter = solana.TimelockConverter{}
		case chainsel.FamilyAptos:
			converter = aptos.NewTimelockConverter()
		default:
			return nil, fmt.Errorf("unsupported chain family %s", fam)
		}

		converters[chainMeta] = converter
	}

	return converters, nil
}
