package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	mcmsapi "github.com/smartcontractkit/chainlink-canton/bindings/generated/mcms/api"
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

// ConvertBatchToChainOperations converts a BatchOperation to Canton-specific timelock operations.
func (t *TimelockConverter) ConvertBatchToChainOperations(
	_ context.Context,
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	_ string, // timelockAddress - not used for Canton (MCMS has built-in timelock)
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	metadataFields, err := resolveAdditionalFieldsMetadata(
		metadata,
		bop,
		uint64(len(bop.Transactions)),
		action,
		false,
	)
	if err != nil {
		return nil, common.Hash{}, err
	}

	calls, callsForHash, allContractIds, err := buildCallsFromBatch(bop)
	if err != nil {
		return nil, common.Hash{}, err
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

	var functionName string
	var opDataHex string
	switch action {
	case types.TimelockActionSchedule:
		functionName, opDataHex, err = scheduleActionData(calls, predecessorHex, saltHex, delay)
	case types.TimelockActionBypass:
		functionName, opDataHex, err = bypassActionData(calls)
	case types.TimelockActionCancel:
		functionName, opDataHex, err = cancelActionData(operationIDStr)
	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported timelock action: %v", action)
	}
	if err != nil {
		return nil, common.Hash{}, err
	}

	// For timelock operations (schedule/cancel/bypass), the target is the MCMS contract itself.
	// Canton expects TargetInstanceAddress in "baseId@partyId" format, not hex.
	// MultisigId is in format "baseId@partyId-role"; extract "baseId@partyId".
	targetInstanceAddress := extractInstanceAddressFromMultisigId(metadataFields.MultisigId)
	op, err := buildTimelockOperation(bop, mcmAddress, targetInstanceAddress, functionName, opDataHex, allContractIds)
	if err != nil {
		return nil, common.Hash{}, err
	}

	return []types.Operation{op}, operationID, nil
}

func OperationID(
	batchOp types.BatchOperation,
	_ types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) (common.Hash, error) {
	_, callsForHash, _, err := buildCallsFromBatch(batchOp)
	if err != nil {
		return common.Hash{}, err
	}

	predecessorHex := hex.EncodeToString(predecessor[:])
	saltHex := hex.EncodeToString(salt[:])
	operationIDStr := HashTimelockOpId(callsForHash, predecessorHex, saltHex)

	operationIDBytes, err := hex.DecodeString(operationIDStr)
	if err != nil || len(operationIDBytes) != 32 {
		return common.Hash{}, fmt.Errorf("invalid operation ID hash: %w", err)
	}

	var operationID common.Hash
	copy(operationID[:], operationIDBytes)

	return operationID, nil
}

// buildCallsFromBatch extracts mcmsapi.TimelockCall, TimelockCallForHash, and all ContractIds from bop.
// Uses tx.To and tx.Data as fallbacks when AdditionalFields are missing or empty.
func buildCallsFromBatch(bop types.BatchOperation) ([]mcmsapi.TimelockCall, []TimelockCallForHash, []string, error) {
	calls := make([]mcmsapi.TimelockCall, 0, len(bop.Transactions))
	callsForHash := make([]TimelockCallForHash, 0, len(bop.Transactions))
	var allContractIds []string
	for _, tx := range bop.Transactions {
		var af AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			if err := json.Unmarshal(tx.AdditionalFields, &af); err != nil {
				return nil, nil, nil, fmt.Errorf("unmarshal transaction additional fields: %w", err)
			}
		}
		targetInstanceAddress := af.TargetInstanceAddress
		if targetInstanceAddress == "" {
			targetInstanceAddress = tx.To
		}
		operationData := af.OperationData
		if operationData == "" && len(tx.Data) > 0 {
			operationData = hex.EncodeToString(tx.Data)
		}
		calls = append(calls, mcmsapi.TimelockCall{
			TargetInstanceAddress: cantontypes.TEXT(targetInstanceAddress),
			FunctionName:          cantontypes.TEXT(af.FunctionName),
			OperationData:         cantontypes.TEXT(operationData),
		})
		callsForHash = append(callsForHash, TimelockCallForHash{
			TargetInstanceAddress: targetInstanceAddress,
			FunctionName:          af.FunctionName,
			OperationData:         operationData,
		})
		allContractIds = append(allContractIds, af.ContractIds...)
	}

	return calls, callsForHash, allContractIds, nil
}

// scheduleActionData returns function name and hex-encoded params for ScheduleBatch.
func scheduleActionData(calls []mcmsapi.TimelockCall, predecessorHex, saltHex string, delay types.Duration) (functionName, opDataHex string, err error) {
	delaySecs := int64(delay.Seconds())
	if delaySecs < 0 {
		delaySecs = 0
	}
	params := mcmsapi.ScheduleBatchParams{
		Calls:       calls,
		Predecessor: cantontypes.TEXT(predecessorHex),
		Salt:        cantontypes.TEXT(saltHex),
		DelaySecs:   cantontypes.INT64(delaySecs),
	}
	opDataHex, err = params.MarshalHex()
	if err != nil {
		return "", "", fmt.Errorf("marshal ScheduleBatchParams: %w", err)
	}

	return "ScheduleBatch", opDataHex, nil
}

// bypassActionData returns function name and hex-encoded params for BypasserExecuteBatch.
func bypassActionData(calls []mcmsapi.TimelockCall) (functionName, opDataHex string, err error) {
	params := mcmsapi.BypasserExecuteBatchParams{Calls: calls}
	opDataHex, err = params.MarshalHex()
	if err != nil {
		return "", "", fmt.Errorf("marshal BypasserExecuteBatchParams: %w", err)
	}

	return "BypasserExecuteBatch", opDataHex, nil
}

// cancelActionData returns function name and hex-encoded params for CancelBatch.
func cancelActionData(operationIDStr string) (functionName, opDataHex string, err error) {
	params := mcmsapi.CancelBatchParams{OpId: cantontypes.TEXT(operationIDStr)}
	opDataHex, err = params.MarshalHex()
	if err != nil {
		return "", "", fmt.Errorf("marshal CancelBatchParams: %w", err)
	}

	return "CancelBatch", opDataHex, nil
}

// extractInstanceAddressFromMultisigId extracts "baseId@partyId" from "baseId@partyId-role".
// MultisigId format: "baseId@partyId-role" (e.g., "mcms-abc123@party::hash-proposer")
// Returns: "baseId@partyId" (e.g., "mcms-abc123@party::hash")
func extractInstanceAddressFromMultisigId(multisigId string) string {
	// Find the last dash that separates the role suffix
	lastDash := strings.LastIndex(multisigId, "-")
	if lastDash == -1 {
		return multisigId
	}

	return multisigId[:lastDash]
}

// buildTimelockOperation builds a single types.Operation for the given timelock action.
// allContractIds are the target contract IDs from the batch (for ExecuteOp TargetCids); mcmAddress is included as TargetCid.
func buildTimelockOperation(bop types.BatchOperation, mcmAddress, targetInstanceAddress, functionName, opDataHex string, allContractIds []string) (types.Operation, error) {
	opAdditionalFields := AdditionalFields{
		TargetInstanceAddress: targetInstanceAddress,
		FunctionName:          functionName,
		OperationData:         opDataHex,
		TargetCid:             mcmAddress,
		ContractIds:           allContractIds,
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	if err != nil {
		return types.Operation{}, fmt.Errorf("marshal operation additional fields: %w", err)
	}

	return types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction: types.Transaction{
			To:               mcmAddress,
			Data:             []byte{0x00}, // placeholder for validators; Canton uses AdditionalFields.OperationData
			AdditionalFields: opAdditionalFieldsBytes,
		},
	}, nil
}
