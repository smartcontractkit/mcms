package aptos

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for EVM chains for accessing the RBACTimelock contract
type TimelockExecutor struct {
	TimelockInspector
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client aptos.AptosRpcClient, auth aptos.TransactionSigner) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: *NewTimelockInspector(client),
		client:            client,
		auth:              auth,
		bindingFn:         mcms.Bind,
	}
}

func (t *TimelockExecutor) Execute(
	_ context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	mcmsAddress, timelockErr := hexToAddress(timelockAddress)
	if timelockErr != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse Timelock address %q: %w", timelockAddress, timelockErr)
	}
	mcmsBinding := t.bindingFn(mcmsAddress, t.client)
	opts := &bind.TransactOpts{Signer: t.auth}

	targets := make([]aptos.AccountAddress, len(bop.Transactions))
	moduleNames := make([]string, len(bop.Transactions))
	functionNames := make([]string, len(bop.Transactions))
	datas := make([][]byte, len(bop.Transactions))

	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		toAddress, err := hexToAddress(tx.To)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to parse To address %q: %w", tx.To, err)
		}

		targets[i] = toAddress
		moduleNames[i] = additionalFields.ModuleName
		functionNames[i] = additionalFields.Function
		datas[i] = tx.Data
	}

	tx, err := mcmsBinding.MCMS().TimelockExecuteBatch(opts, targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes())
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute batch: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Hash,
		ChainFamily: chain_selectors.FamilyAptos,
		RawData:     tx,
	}, nil
}
