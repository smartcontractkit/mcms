package aptos

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aptos-labs/aptos-go-sdk"

	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for Aptos, for accessing the MCMS-Timelock
type TimelockExecutor struct {
	TimelockInspector
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner
}

// NewTimelockExecutor creates a new TimelockExecutor that talks to a standard MCMS contract.
func NewTimelockExecutor(client aptos.AptosRpcClient, auth aptos.TransactionSigner) *TimelockExecutor {
	return NewTimelockExecutorWithMCMSType(client, auth, MCMSTypeRegular)
}

// NewTimelockExecutorWithMCMSType creates a TimelockExecutor that talks to either a
// standard MCMS or CurseMCMS contract depending on mcmsType.
func NewTimelockExecutorWithMCMSType(client aptos.AptosRpcClient, auth aptos.TransactionSigner, mcmsType MCMSType) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: *NewTimelockInspectorWithMCMSType(client, mcmsType),
		client:            client,
		auth:              auth,
	}
}

func (t *TimelockExecutor) Execute(
	_ context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	mcmsAddress, timelockErr := hexToAddress(timelockAddress)
	if timelockErr != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse Timelock address %q: %w", timelockAddress, timelockErr)
	}
	contract := t.contractFn(mcmsAddress, t.client)
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

	tx, err := contract.TimelockExecuteBatch(opts, targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes())
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute batch: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Hash,
		ChainFamily: chainsel.FamilyAptos,
		RawData:     tx,
	}, nil
}

func (t TimelockExecutor) Equal(other TimelockExecutor) bool {
	return true
}
