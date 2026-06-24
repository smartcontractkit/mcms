package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

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

// GrantRoles calls grantRole on the RBACTimelock contract for each target address.
func (c *TimelockConfigurer) GrantRoles(
	ctx context.Context,
	timelockAddress string,
	role sdk.TimelockRole,
	addresses []string,
) (types.TransactionResult, error) {
	if !common.IsHexAddress(timelockAddress) {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %s", timelockAddress)
	}

	roleHash, err := role.Hash()
	if err != nil {
		return types.TransactionResult{}, err
	}

	if len(addresses) == 0 {
		return types.TransactionResult{}, fmt.Errorf("addresses must be non-empty")
	}

	timelock := common.HexToAddress(timelockAddress)
	if timelock == (common.Address{}) {
		return types.TransactionResult{}, fmt.Errorf("invalid timelock address: %s", timelockAddress)
	}

	accounts := make([]common.Address, 0, len(addresses))
	for _, address := range addresses {
		if !common.IsHexAddress(address) {
			return types.TransactionResult{}, fmt.Errorf("invalid target address: %s", address)
		}

		account := common.HexToAddress(address)
		if account == (common.Address{}) {
			return types.TransactionResult{}, fmt.Errorf("invalid target address: %s", address)
		}

		accounts = append(accounts, account)
	}

	opts := *c.auth
	opts.Context = ctx

	tl, err := bindings.NewRBACTimelock(timelock, c.client)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to bind RBACTimelock at %s: %w", timelockAddress, err)
	}

	txs := make([]*gethtypes.Transaction, 0, len(accounts))
	for _, account := range accounts {
		tx, err := tl.GrantRole(&opts, [32]byte(roleHash), account)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to grant role %s to %s on %s: %w", role, account.Hex(), timelockAddress, err)
		}

		txs = append(txs, tx)
	}

	hash := ""
	if len(txs) > 0 {
		hash = txs[0].Hash().Hex()
	}

	return types.TransactionResult{
		Hash:        hash,
		ChainFamily: chainsel.FamilyEVM,
		RawData:     txs,
	}, nil
}
