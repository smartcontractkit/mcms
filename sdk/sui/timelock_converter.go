package sui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const (
	TimelockActionSchedule = "timelock_schedule_batch"
	TimelockActionCancel   = "timelock_cancel"
	TimelockActionBypass   = "timelock_bypasser_execute_batch"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

type TimelockConverter struct {
	client        sui.ISuiAPI
	signer        bindutils.SuiSigner
	mcmsPackageId string
	mcms          module_mcms.IMcms
}

func NewTimelockConverter(client sui.ISuiAPI, signer bindutils.SuiSigner, mcmsPackageId string) (*TimelockConverter, error) {
	mcms, err := module_mcms.NewMcms(mcmsPackageId, client)
	if err != nil {
		return nil, err
	}

	return &TimelockConverter{
		client:        client,
		signer:        signer,
		mcmsPackageId: mcmsPackageId,
		mcms:          mcms,
	}, nil
}

// We need somehow to create an mcms tx that contains the timelock command. The execute will then create a PTB with execute and the command coming from the proposal, which has the timelock command
// This thing should just return the part of the PTB calling the correspinding dispatch function
func (t *TimelockConverter) ConvertBatchToChainOperations(
	ctx context.Context,
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	timelockAddress string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	// Extract transaction data from batch operation
	targets := make([][]byte, len(bop.Transactions))
	moduleNames := make([]string, len(bop.Transactions))
	functionNames := make([]string, len(bop.Transactions))
	datas := make([][]byte, len(bop.Transactions))
	tags := make([]string, 0, len(bop.Transactions))

	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}

		// Convert Sui address properly using AddressFromHex to ensure correct padding
		targetAddr, err := AddressFromHex(tx.To)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to parse target address %q: %w", tx.To, err)
		}
		targets[i] = targetAddr.Bytes()
		moduleNames[i] = additionalFields.ModuleName
		functionNames[i] = additionalFields.Function
		datas[i] = tx.Data
		tags = append(tags, tx.Tags...)
	}

	// Create transaction based on action type
	var function string
	var data []byte
	var err error
	switch action {
	case types.TimelockActionSchedule:
		function = TimelockActionSchedule
		data, err = SerializeTimelockScheduleBatch(targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes(), uint64(delay.Seconds()))
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to serialize timelock schedule batch: %w", err)
		}
	case types.TimelockActionCancel:
		function = TimelockActionCancel
	case types.TimelockActionBypass:
		function = TimelockActionBypass
		data, err = SerializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to serialize timelock bypasser execute batch: %w", err)
		}
	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported timelock action: %v", action)
	}

	// Create the transaction
	tx, err := NewTransaction(
		"mcms", // can only be mcms
		function,
		t.mcmsPackageId, // can only call itself
		data,
		"MCMS",
		tags,
	)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction:   tx,
	}

	operationID, err := HashOperationBatch(targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes())
	return []types.Operation{op}, operationID, nil
}

// HashOperationBatch calculates the hash of a batch operation (public function for compatibility)
func HashOperationBatch(targets [][]byte, moduleNames, functionNames []string, datas [][]byte, predecessor, salt []byte) (common.Hash, error) {
	// Create a hash based on the operation parameters
	hasher := crypto.NewKeccakState()

	// Write number of targets
	hasher.Write([]byte(fmt.Sprintf("%d", len(targets))))

	// Write each target, module, function, and data
	for i, target := range targets {
		hasher.Write(target)
		hasher.Write([]byte(moduleNames[i]))
		hasher.Write([]byte(functionNames[i]))
		hasher.Write(datas[i])
	}

	// Write predecessor and salt
	hasher.Write(predecessor)
	hasher.Write(salt)

	var hash common.Hash
	hasher.Read(hash[:])
	return hash, nil
}
