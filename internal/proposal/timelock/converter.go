package timelock

import (
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/core/proposal/timelock"
	evm_timelock "github.com/smartcontractkit/mcms/internal/evm/proposal/timelock"
)

// ToChainOperation converts a batch of chain operations to a single types.ChainOperation
func ToChainOperation(
	t timelock.BatchChainOperation,
	timelockAddress common.Address,
	minDelay string,
	operation timelock.TimelockOperationType,
	predecessor common.Hash,
) (mcms.ChainOperation, common.Hash, error) {
	chainFamily, err := chain_selectors.GetSelectorFamily(uint64(t.ChainSelector))
	if err != nil {
		return mcms.ChainOperation{}, common.Hash{}, err
	}

	var converter timelock.TimelockConverter

	switch chainFamily {
	case chain_selectors.FamilyEVM:
		converter = &evm_timelock.TimelockConverterEVM{}
	default:
		return mcms.ChainOperation{}, common.Hash{}, core.NewUnknownChainSelectorFamilyError(uint64(t.ChainSelector), chainFamily)
	}

	return converter.ConvertBatchToChainOperation(t, timelockAddress, minDelay, operation, predecessor)
}
