package evm

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for EVM chains for accessing the RBACTimelock contract
type TimelockExecutor struct {
	TimelockInspector
	client ContractDeployBackend
	auth   *bind.TransactOpts
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client ContractDeployBackend, auth *bind.TransactOpts) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: *NewTimelockInspector(client),
		client:            client,
		auth:              auth,
	}
}

// NOTE: a CallProxy can be used to execute the calls by replacing the
// timelock address with the proxy address.
func (t *TimelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(timelockAddress), t.client)
	if err != nil {
		return types.TransactionResult{}, err
	}

	calls := make([]bindings.RBACTimelockCall, len(bop.Transactions))
	for i, tx := range bop.Transactions {
		// Unmarshal the AdditionalFields from the operation
		var additionalFields AdditionalFields
		if err = json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return types.TransactionResult{}, err
		}

		calls[i] = bindings.RBACTimelockCall{
			Target: common.HexToAddress(tx.To),
			Data:   tx.Data,
			Value:  additionalFields.Value,
		}
	}

	opts := *t.auth
	opts.Context = ctx

	tx, err := timelock.ExecuteBatch(&opts, calls, predecessor, salt)
	if err != nil {
		return types.TransactionResult{}, err
	}

	return types.TransactionResult{
		Hash:        tx.Hash().Hex(),
		ChainFamily: chain_selectors.FamilyEVM,
		RawData:     tx,
	}, nil
}
