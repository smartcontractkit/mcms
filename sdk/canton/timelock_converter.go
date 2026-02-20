package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

// TimelockConverter converts Canton timelock batch operations to chain operations.
type TimelockConverter struct{}

// NewTimelockConverter creates a new TimelockConverter.
func NewTimelockConverter() *TimelockConverter {
	return &TimelockConverter{}
}

func (t *TimelockConverter) ConvertBatchToChainOperations(
	ctx context.Context,
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	_ string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	_ = ctx

	var meta AdditionalFieldsMetadata
	if err := json.Unmarshal(metadata.AdditionalFields, &meta); err != nil {
		return nil, common.Hash{}, fmt.Errorf("unmarshal metadata additional fields: %w", err)
	}

	// Build bindings TimelockCall slice and the same for hash input
	calls := make([]mcms.TimelockCall, 0, len(bop.Transactions))
	callsForHash := make([]TimelockCallForHash, 0, len(bop.Transactions))
	for _, tx := range bop.Transactions {
		var af AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &af); err != nil {
			return nil, common.Hash{}, fmt.Errorf("unmarshal transaction additional fields: %w", err)
		}
		calls = append(calls, mcms.TimelockCall{
			TargetInstanceId: cantontypes.TEXT(af.TargetInstanceId),
			FunctionName:     cantontypes.TEXT(af.FunctionName),
			OperationData:    cantontypes.TEXT(af.OperationData),
		})
		callsForHash = append(callsForHash, TimelockCallForHash{
			TargetInstanceId: af.TargetInstanceId,
			FunctionName:     af.FunctionName,
			OperationData:    af.OperationData,
		})
	}

	predecessorHex := hex.EncodeToString(predecessor[:])
	saltHex := hex.EncodeToString(salt[:])

	operationIDStr := HashTimelockOpId(callsForHash, predecessorHex, saltHex)
	operationIDBytes, err := hex.DecodeString(operationIDStr)
	if err != nil || len(operationIDBytes) != 32 {
		return nil, common.Hash{}, fmt.Errorf("invalid operation ID hash: %w", err)
	}
	var operationID common.Hash
	copy(operationID[:], operationIDBytes)

	switch action {
	case types.TimelockActionSchedule:
		delaySecs := int64(delay.Duration.Seconds())
		if delaySecs < 0 {
			delaySecs = 0
		}
		scheduleParams := mcms.ScheduleBatchParams{
			Calls:       calls,
			Predecessor: cantontypes.TEXT(predecessorHex),
			Salt:        cantontypes.TEXT(saltHex),
			DelaySecs:   cantontypes.INT64(delaySecs),
		}
		opDataHex, err := scheduleParams.MarshalHex()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("marshal ScheduleBatchParams: %w", err)
		}

		targetInstanceId := meta.InstanceId
		if targetInstanceId == "" {
			targetInstanceId = meta.MultisigId
		}
		opAdditionalFields := AdditionalFields{
			TargetInstanceId: targetInstanceId,
			FunctionName:     "ScheduleBatch",
			OperationData:    opDataHex,
			TargetCid:        mcmAddress,
			ContractIds:      []string{mcmAddress},
		}
		opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("marshal operation additional fields: %w", err)
		}

		op := types.Operation{
			ChainSelector: bop.ChainSelector,
			Transaction: types.Transaction{
				To:               mcmAddress,
				Data:             []byte{},
				AdditionalFields: opAdditionalFieldsBytes,
			},
		}
		return []types.Operation{op}, operationID, nil

	case types.TimelockActionBypass:
		// BypasserExecuteBatch: only calls (no predecessor/salt in params)
		bypassParams := mcms.BypasserExecuteBatchParams{
			Calls: calls,
		}
		opDataHex, err := bypassParams.MarshalHex()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("marshal BypasserExecuteBatchParams: %w", err)
		}
		targetInstanceId := meta.InstanceId
		if targetInstanceId == "" {
			targetInstanceId = meta.MultisigId
		}
		opAdditionalFields := AdditionalFields{
			TargetInstanceId: targetInstanceId,
			FunctionName:     "BypasserExecuteBatch",
			OperationData:    opDataHex,
			TargetCid:        mcmAddress,
			ContractIds:      []string{mcmAddress},
		}
		opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("marshal operation additional fields: %w", err)
		}
		op := types.Operation{
			ChainSelector: bop.ChainSelector,
			Transaction: types.Transaction{
				To:               mcmAddress,
				Data:             []byte{},
				AdditionalFields: opAdditionalFieldsBytes,
			},
		}
		return []types.Operation{op}, operationID, nil

	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported timelock action: %v", action)
	}
}
