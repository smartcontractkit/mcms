package timelock

import (
	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// ToChainOperation converts a batch of chain operations to a single types.ChainOperation
func ToChainOperation(
	t types.BatchChainOperation,
	timelockAddress string,
	minDelay string,
	operation types.TimelockAction,
	predecessor common.Hash,
) (types.ChainOperation, common.Hash, error) {
	chainFamily, err := types.GetChainSelectorFamily(t.ChainSelector)
	if err != nil {
		return types.ChainOperation{}, common.Hash{}, err
	}

	var converter sdk.TimelockConverter

	switch chainFamily {
	case cselectors.FamilyEVM:
		converter = &evm.TimelockConverterEVM{}
	}

	return converter.ConvertBatchToChainOperation(t, timelockAddress, minDelay, operation, predecessor)
}
