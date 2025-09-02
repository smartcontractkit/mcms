package sui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for Sui, for accessing the MCMS-Timelock
type TimelockExecutor struct {
	TimelockInspector
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	registryObj   string
	accountObj    string
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client sui.ISuiAPI, signer bindutils.SuiSigner, mcmsPackageId string, registryObj string, accountObj string) (*TimelockExecutor, error) {
	timelockInspector, err := NewTimelockInspector(client, signer, mcmsPackageId)
	if err != nil {
		return nil, err
	}

	return &TimelockExecutor{
		TimelockInspector: *timelockInspector,
		client:            client,
		signer:            signer,
		mcmsPackageId:     mcmsPackageId,
		registryObj:       registryObj,
		accountObj:        accountObj,
	}, nil
}

func (t *TimelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	targets := make([]string, len(bop.Transactions))
	moduleNames := make([]string, len(bop.Transactions))
	functionNames := make([]string, len(bop.Transactions))
	datas := make([][]byte, len(bop.Transactions))

	calls := make([]Call, 0, len(bop.Transactions))
	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		targets[i] = tx.To
		moduleNames[i] = additionalFields.ModuleName
		functionNames[i] = additionalFields.Function
		datas[i] = tx.Data

		// Convert Sui address properly using AddressFromHex to ensure correct padding
		targetAddress, err := AddressFromHex(tx.To)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to parse target address %q: %w", tx.To, err)
		}

		calls = append(calls, Call{
			StateObj:     additionalFields.StateObj,
			Target:       targetAddress.Bytes(),
			ModuleName:   additionalFields.ModuleName,
			FunctionName: additionalFields.Function,
			Data:         tx.Data,
		})
	}

	timelockObj := bind.Object{Id: timelockAddress}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	timelockExecuteCall, err := t.mcms.Encoder().TimelockExecuteBatch(
		timelockObj,
		clockObj,
		targets,
		moduleNames,
		functionNames,
		datas,
		predecessor.Bytes(),
		salt.Bytes(),
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute batch: %w", err)
	}

	opts := &bind.CallOpts{Signer: t.signer, WaitForExecution: true}

	ptb := transaction.NewTransaction()
	executeCallback, err := t.mcms.AppendPTB(ctx, opts, ptb, timelockExecuteCall)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("building PTB for execute call: %w", err)
	}

	err = AppendPTBFromExecutingCallbackParams(ctx, t.client, t.mcms, ptb, t.mcmsPackageId, executeCallback, calls, t.registryObj, t.accountObj)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("extending PTB from executing callback params: %w", err)
	}

	// Execute the complete PTB with every call
	tx, err := bind.ExecutePTB(ctx, opts, t.client, ptb)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("op execution with PTB failed: %w", err)
	}

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: cselectors.FamilySui,
		RawData:     tx,
	}, nil
}
