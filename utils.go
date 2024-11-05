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
	batchOps types.BatchChainOperation,
	timelockAddr string,
	delay string,
	action types.TimelockAction,
	predecessor common.Hash,
) (types.ChainOperation, common.Hash, error) {
	chainFamily, err := types.GetChainSelectorFamily(batchOps.ChainSelector)
	if err != nil {
		return types.ChainOperation{}, common.Hash{}, err
	}

	var converter sdk.TimelockConverter
	switch chainFamily {
	case cselectors.FamilyEVM:
		converter = &evm.TimelockConverterEVM{}
	}

	return converter.ConvertBatchToChainOperation(
		batchOps, timelockAddr, delay, action, predecessor,
	)
}
