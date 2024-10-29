package timelock

import (
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/core"
	mcmsTypes "github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
	timelockTypes "github.com/smartcontractkit/mcms/pkg/proposal/timelock/types"

	"github.com/smartcontractkit/mcms/sdk/evm"
)

// TimelockConverter converts a batch of chain operations to an types.ChainOperation
type TimelockConverter interface {
	ConvertBatchToChainOperation(
		t timelockTypes.BatchChainOperation,
		timelockAddress common.Address,
		minDelay string,
		operation timelockTypes.TimelockOperationType,
		predecessor common.Hash,
	) (mcmsTypes.ChainOperation, common.Hash, error)
}

// ToChainOperation converts a batch of chain operations to a single types.ChainOperation
func ToChainOperation(
	t timelockTypes.BatchChainOperation,
	timelockAddress common.Address,
	minDelay string,
	operation timelockTypes.TimelockOperationType,
	predecessor common.Hash,
) (mcmsTypes.ChainOperation, common.Hash, error) {
	chainFamily, err := chain_selectors.GetSelectorFamily(uint64(t.ChainIdentifier))
	if err != nil {
		return mcmsTypes.ChainOperation{}, common.Hash{}, err
	}

	var converter TimelockConverter

	switch chainFamily {
	case chain_selectors.FamilyEVM:
		converter = &evm.TimelockConverterEVM{}
	default:
		return mcmsTypes.ChainOperation{}, common.Hash{}, core.NewUnknownChainSelectorFamilyError(uint64(t.ChainIdentifier), chainFamily)
	}

	return converter.ConvertBatchToChainOperation(t, timelockAddress, minDelay, operation, predecessor)
}
