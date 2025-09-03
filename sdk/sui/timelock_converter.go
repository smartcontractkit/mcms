package sui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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
}

func NewTimelockConverter() (*TimelockConverter, error) {

	return &TimelockConverter{}, nil
}

// We need somehow to create an mcms tx that contains the timelock command. The execute will then create a PTB with execute and the command coming from the proposal, which has the timelock command
// This thing should just return the part of the PTB calling the corresponding dispatch function
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
	stateObjs := make([]string, len(bop.Transactions))
	targets := make([][]byte, len(bop.Transactions))
	moduleNames := make([]string, len(bop.Transactions))
	functionNames := make([]string, len(bop.Transactions))
	datas := make([][]byte, len(bop.Transactions))
	tags := make([]string, 0, len(bop.Transactions))

	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return []types.Operation{}, common.Hash{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

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
		stateObjs[i] = additionalFields.StateObj
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
		data, err = serializeTimelockScheduleBatch(targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes(), uint64(delay.Milliseconds()))
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to serialize timelock schedule batch: %w", err)
		}
	case types.TimelockActionCancel:
		// TODO: Implement cancellation flow
		function = TimelockActionCancel
	case types.TimelockActionBypass:
		function = TimelockActionBypass
		data, err = serializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to serialize timelock bypasser execute batch: %w", err)
		}
	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported timelock action: %v", action)
	}

	// Create the transaction
	tx, err := NewTransactionWithManyStateObj(
		"mcms", // can only be mcms
		function,
		additionalFieldsMetadata.McmsPackageID, // can only call itself
		data,
		"MCMS",
		tags,
		timelockAddress,
		stateObjs,
	)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction:   tx,
	}

	operationID, hashErr := HashOperationBatch(targets, moduleNames, functionNames, datas, predecessor.Bytes(), salt.Bytes())
	if hashErr != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to hash operation batch: %w", hashErr)
	}

	return []types.Operation{op}, operationID, nil
}

// HashOperationBatch calculates the hash of a batch operation using BCS serialization
func HashOperationBatch(targets [][]byte, moduleNames, functionNames []string, datas [][]byte, predecessor, salt []byte) (common.Hash, error) {
	callsBytes, err := bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.Uleb128(uint32(len(targets)))

		for i := range targets {
			ser.FixedBytes(targets[i])
			ser.WriteString(moduleNames[i])
			ser.WriteString(functionNames[i])

			ser.WriteBytes(datas[i])
		}
	})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to BCS serialize calls: %w", err)
	}

	var packed []byte
	packed = append(packed, callsBytes...)
	packed = append(packed, predecessor...)
	packed = append(packed, salt...)

	hash := crypto.Keccak256Hash(packed)

	return hash, nil
}
