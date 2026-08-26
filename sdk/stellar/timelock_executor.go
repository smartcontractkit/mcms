package stellar

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	stellarrpc "github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

type TimelockExecutor struct {
	*TimelockInspector
	invoker bindings.Invoker
	caller  string
}

func NewTimelockExecutor(client *stellarrpc.Client, auth bindings.Signer, selector uint64, caller string) (*TimelockExecutor, error) {
	invoker, err := NewInvoker(client, auth, selector)
	if err != nil {
		return nil, err
	}

	return &TimelockExecutor{TimelockInspector: NewTimelockInspectorFromInvoker(invoker), invoker: invoker, caller: caller}, nil
}

// NewTimelockExecutorWithNetworkPassphrase creates an executor with an
// explicit network passphrase supplied by the deployment framework.
func NewTimelockExecutorWithNetworkPassphrase(client *stellarrpc.Client, auth bindings.Signer, passphrase, caller string) (*TimelockExecutor, error) {
	invoker, err := NewInvokerWithNetworkPassphrase(client, auth, passphrase)
	if err != nil {
		return nil, err
	}

	return &TimelockExecutor{TimelockInspector: NewTimelockInspectorFromInvoker(invoker), invoker: invoker, caller: caller}, nil
}

func (e *TimelockExecutor) Execute(
	ctx context.Context,
	batch types.BatchOperation,
	address string,
	predecessor common.Hash,
	salt common.Hash,
) (types.TransactionResult, error) {
	if e == nil || e.invoker == nil {
		return types.TransactionResult{},
			fmt.Errorf("stellar timelock invoker is nil")
	}

	calls, err := stellarCallValues(batch.Transactions)
	if err != nil {
		return types.TransactionResult{}, err
	}

	data, err := EncodeSorobanInvokePayload(
		"execute_batch",
		[]xdr.ScVal{
			calls,
			scval.Bytes32ToScVal([32]byte(predecessor)),
			scval.Bytes32ToScVal([32]byte(salt)),
		},
	)
	if err != nil {
		return types.TransactionResult{}, err
	}

	payload, err := DecodeSorobanInvokePayload(data)
	if err != nil {
		return types.TransactionResult{}, err
	}

	_, err = e.invoker.InvokeContract(
		ctx,
		address,
		"execute_batch",
		payload.Args,
	)
	if err != nil {
		originalErr := fmt.Errorf(
			"stellar timelock execute_batch: %w",
			err,
		)

		var failedTransaction *types.Transaction
		if len(batch.Transactions) == 1 {
			failedTransaction = &batch.Transactions[0]
		}

		return types.TransactionResult{}, newExecutionError(
			failedTransaction,
			originalErr,
			map[string]contractKind{
				canonicalContractID(address): contractKindTimelock,
			},
		)
	}

	return types.NewTransactionResult(
		"",
		nil,
		"stellar",
	), nil
}

func stellarCallValues(txs []types.Transaction) (xdr.ScVal, error) {
	if len(txs) == 0 {
		return xdr.ScVal{}, fmt.Errorf("empty Stellar timelock batch")
	}
	calls := make([]xdr.ScVal, 0, len(txs))
	for _, tx := range txs {
		p, err := DecodeSorobanInvokePayload(tx.Data)
		if err != nil {
			return xdr.ScVal{}, err
		}
		call, err := scval.BuildStructScVal(map[string]xdr.ScVal{"target": scval.AddressToScVal(tx.To), "function": scval.SymbolToScVal(p.Function), "args_xdr": scval.BytesToScVal(p.ArgsXDR)})
		if err != nil {
			return xdr.ScVal{}, err
		}
		calls = append(calls, call)
	}

	return scval.BuildStructScVal(map[string]xdr.ScVal{"inner": scval.VecToScVal(calls)})
}
