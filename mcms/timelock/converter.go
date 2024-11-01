package timelock

import (
	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/core/proposal/timelock"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// ToChainOperation converts a batch of chain operations to a single types.ChainOperation
func ToChainOperation(
	t types.BatchChainOperation,
	timelockAddress common.Address,
	minDelay string,
	operation types.TimelockAction,
	predecessor common.Hash,
) (types.ChainOperation, common.Hash, error) {
	chainFamily, err := types.GetChainSelectorFamily(t.ChainSelector)
	if err != nil {
		return types.ChainOperation{}, common.Hash{}, err
	}

	var converter timelock.TimelockConverter

	switch chainFamily {
	case cselectors.FamilyEVM:
		converter = &evm.TimelockConverterEVM{}
	}

	return converter.ConvertBatchToChainOperation(t, timelockAddress, minDelay, operation, predecessor)
}
