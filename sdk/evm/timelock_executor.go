package evm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

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
	opts := *t.auth
	opts.Context = ctx

	timelockAddr := common.HexToAddress(timelockAddress)
	timelock, err := bindings.NewRBACTimelock(timelockAddr, t.client)
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

	// Pre-pack calldata so we always have something to return on failure.
	// This is useful if the tx fails to give the user the calldata to retry or simulate.
	txPreview, err := buildTimelockExecuteTxData(&opts, timelockAddr, calls, predecessor, salt)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to build execute call data: %w", err)
	}

	tx, err := timelock.ExecuteBatch(&opts, calls, predecessor, salt)
	if err != nil {
		timelockCallData := txPreview.Data()
		execErr := BuildExecutionError(ctx, err, txPreview, &opts, timelockAddr, t.client, timelockAddr, timelockCallData)

		return types.TransactionResult{
			ChainFamily: chain_selectors.FamilyEVM,
		}, execErr
	}

	return types.TransactionResult{
		Hash:        tx.Hash().Hex(),
		ChainFamily: chain_selectors.FamilyEVM,
		RawData:     tx,
	}, nil
}

// buildTimelockExecuteTxData packs call data for RBACTimelock.executeBatch(...)
func buildTimelockExecuteTxData(
	opts *bind.TransactOpts,
	timelockAddr common.Address,
	calls []bindings.RBACTimelockCall,
	predecessor [32]byte,
	salt [32]byte,
) (*gethtypes.Transaction, error) {
	abi, err := bindings.RBACTimelockMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	data, err := abi.Pack("executeBatch", calls, predecessor, salt)
	if err != nil {
		return nil, err
	}
	tx := buildUnsignedTxFromOpts(opts, timelockAddr, data)

	return tx, nil
}
