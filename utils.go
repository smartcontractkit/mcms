package mcms

import (
	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// BatchToChainOperation converts a batch of chain operations to a single types.ChainOperation for
// different chains
func BatchToChainOperation(
	bops types.BatchOperation,
	timelockAddr string,
	delay string,
	action types.TimelockAction,
	predecessor common.Hash,
) (types.Operation, common.Hash, error) {
	chainFamily, err := types.GetChainSelectorFamily(bops.ChainSelector)
	if err != nil {
		return types.Operation{}, common.Hash{}, err
	}

	var converter sdk.TimelockConverter
	switch chainFamily {
	case cselectors.FamilyEVM:
		converter = &evm.TimelockConverterEVM{}
	}

	return converter.ConvertBatchToChainOperation(
		bops, timelockAddr, delay, action, predecessor,
	)
}
