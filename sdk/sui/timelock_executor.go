package sui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

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
	mcms          module_mcms.IMcms
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client sui.ISuiAPI, signer bindutils.SuiSigner, mcmsPackageId string) (*TimelockExecutor, error) {
	mcms, err := module_mcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}

	timelockInspector, err := NewTimelockInspector(client, signer, mcmsPackageId)
	if err != nil {
		return nil, err
	}

	return &TimelockExecutor{
		TimelockInspector: *timelockInspector,
		client:            client,
		signer:            signer,
		mcmsPackageId:     mcmsPackageId,
		mcms:              mcms,
	}, nil
}

func (t *TimelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	opts := &bind.CallOpts{Signer: t.signer}

	targets := make([]string, len(bop.Transactions))
	moduleNames := make([]string, len(bop.Transactions))
	functionNames := make([]string, len(bop.Transactions))
	datas := make([][]byte, len(bop.Transactions))

	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return types.TransactionResult{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		targets[i] = tx.To
		moduleNames[i] = additionalFields.ModuleName
		functionNames[i] = additionalFields.Function
		datas[i] = tx.Data
	}

	timelockObj := bind.Object{Id: timelockAddress}
	clockObj := bind.Object{Id: "0x6"} // Clock object ID in Sui

	tx, err := t.mcms.TimelockExecuteBatch(
		ctx,
		opts,
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

	return types.TransactionResult{
		Hash:        tx.Digest,
		ChainFamily: chain_selectors.FamilySui,
		RawData:     tx,
	}, nil
}
