package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

// TimelockConfigurer configures timelock parameters on EVM chains.
type TimelockConfigurer struct {
	TimelockInspector
	client ContractDeployBackend
	auth   *bind.TransactOpts
}

// NewTimelockConfigurer creates a new TimelockConfigurer for EVM chains.
func NewTimelockConfigurer(client ContractDeployBackend, auth *bind.TransactOpts) *TimelockConfigurer {
	return &TimelockConfigurer{
		TimelockInspector: *NewTimelockInspector(client),
		client:            client,
		auth:              auth,
	}
}

// UpdateDelay calls updateDelay on the RBACTimelock contract to change the minimum delay.
func (c *TimelockConfigurer) UpdateDelay(
	ctx context.Context, timelockAddress string, newDelay uint64,
) (types.TransactionResult, error) {
	opts := *c.auth
	opts.Context = ctx

	tl, err := bindings.NewRBACTimelock(common.HexToAddress(timelockAddress), c.client)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to bind RBACTimelock at %s: %w", timelockAddress, err)
	}

	tx, err := tl.UpdateDelay(&opts, new(big.Int).SetUint64(newDelay))
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to update delay on %s: %w", timelockAddress, err)
	}

	return types.TransactionResult{
		Hash:        tx.Hash().Hex(),
		ChainFamily: chainsel.FamilyEVM,
		RawData:     tx,
	}, nil
}
