package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

// TimelockConverter converts batch operations to Canton-specific timelock operations.
// Canton's MCMS contract has built-in timelock functionality, so operations are
// self-dispatched to the MCMS contract itself.
type TimelockConverter struct{}

// NewTimelockConverter creates a new TimelockConverter for Canton.
func NewTimelockConverter() *TimelockConverter {
	return &TimelockConverter{}
}

// TimelockCall represents a single call within a timelock batch.
// Matches the Daml TimelockCall type in MCMS.Types.
type TimelockCall struct {
	TargetInstanceId string `json:"targetInstanceId"`
	FunctionName     string `json:"functionName"`
	OperationData    string `json:"operationData"`
}

// ConvertBatchToChainOperations converts a BatchOperation to Canton-specific timelock operations.
// For Canton, timelock operations are self-dispatched to the MCMS contract with encoded parameters.
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
	// Extract Canton-specific metadata
	var metadataFields AdditionalFieldsMetadata
	if err := json.Unmarshal(metadata.AdditionalFields, &metadataFields); err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to unmarshal metadata additional fields: %w", err)
	}

	// Convert transactions to TimelockCalls
	calls := make([]TimelockCall, len(bop.Transactions))
	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
				return nil, common.Hash{}, fmt.Errorf("failed to unmarshal transaction additional fields: %w", err)
			}
		}

		// Use TargetInstanceId from AdditionalFields, or fall back to tx.To
		targetInstanceId := additionalFields.TargetInstanceId
		if targetInstanceId == "" {
			targetInstanceId = tx.To
		}

		// Use FunctionName from AdditionalFields
		functionName := additionalFields.FunctionName

		// Use OperationData from AdditionalFields, or hex-encode tx.Data
		operationData := additionalFields.OperationData
		if operationData == "" && len(tx.Data) > 0 {
			operationData = hex.EncodeToString(tx.Data)
		}

		calls[i] = TimelockCall{
			TargetInstanceId: targetInstanceId,
			FunctionName:     functionName,
			OperationData:    operationData,
		}
	}

	// Compute operation ID using Canton's hash scheme
	predecessorHex := hex.EncodeToString(predecessor[:])
	saltHex := hex.EncodeToString(salt[:])
	operationID := HashTimelockOpId(calls, predecessorHex, saltHex)

	// Build the timelock operation based on action type
	var timelockFunctionName string
	var operationDataEncoded string
	var err error

	switch action {
	case types.TimelockActionSchedule:
		params := mcms.ScheduleBatchParams{
			Calls:       toMCMSTimelockCalls(calls),
			Predecessor: cantontypes.TEXT(predecessorHex),
			Salt:        cantontypes.TEXT(saltHex),
			DelaySecs:   cantontypes.INT64(delay.Seconds()),
		}
		operationDataEncoded, err = params.MarshalHex()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to encode ScheduleBatchParams: %w", err)
		}
		timelockFunctionName = "ScheduleBatch"

	case types.TimelockActionBypass:
		params := mcms.BypasserExecuteBatchParams{
			Calls: toMCMSTimelockCalls(calls),
		}
		operationDataEncoded, err = params.MarshalHex()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to encode BypasserExecuteBatchParams: %w", err)
		}
		timelockFunctionName = "BypasserExecuteBatch"

	case types.TimelockActionCancel:
		params := mcms.CancelBatchParams{
			OpId: cantontypes.TEXT(hex.EncodeToString(operationID[:])),
		}
		operationDataEncoded, err = params.MarshalHex()
		if err != nil {
			return nil, common.Hash{}, fmt.Errorf("failed to encode CancelBatchParams: %w", err)
		}
		timelockFunctionName = "CancelBatch"

	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported timelock action: %s", action)
	}

	// Collect all target CIDs from the original batch transactions
	// These are needed for BypasserExecuteBatch and other operations that call external contracts
	var allContractIds []string
	for _, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err == nil {
				allContractIds = append(allContractIds, additionalFields.ContractIds...)
			}
		}
	}

	// Build the Canton operation additional fields
	opAdditionalFields := AdditionalFields{
		TargetInstanceId: metadataFields.InstanceId, // Self-dispatch to MCMS (uses instanceId, not multisigId)
		FunctionName:     timelockFunctionName,
		OperationData:    operationDataEncoded,
		TargetCid:        mcmAddress,     // MCMS contract ID
		ContractIds:      allContractIds, // Pass through original target contract IDs
	}

	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to marshal operation additional fields: %w", err)
	}

	// Create the operation
	// Note: Data must be non-empty for validation, but Canton uses AdditionalFields.OperationData
	// We use a placeholder byte to satisfy the validator
	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction: types.Transaction{
			To:               mcmAddress,
			Data:             []byte{0x00}, // Placeholder - actual data is in AdditionalFields.OperationData
			AdditionalFields: opAdditionalFieldsBytes,
		},
	}

	return []types.Operation{op}, operationID, nil
}

// HashTimelockOpId computes the operation ID for a timelock batch.
// Matches Canton's Crypto.daml hashTimelockOpId function.
func HashTimelockOpId(calls []TimelockCall, predecessor, salt string) common.Hash {
	// Encode each call
	var encoded string
	for _, call := range calls {
		encoded += asciiToHex(call.TargetInstanceId)
		encoded += asciiToHex(call.FunctionName)
		encoded += encodeOperationData(call.OperationData)
	}

	// Combine with predecessor and salt (convert ASCII to hex)
	encoded += asciiToHex(predecessor)
	encoded += asciiToHex(salt)

	// Decode hex string and hash
	data, err := hex.DecodeString(encoded)
	if err != nil {
		// If encoding fails, return empty hash (should not happen with valid input)
		return common.Hash{}
	}

	return crypto.Keccak256Hash(data)
}

// encodeOperationData encodes operation data for hashing.
// If it's valid hex, treat it as raw bytes. Otherwise, hex-encode as ASCII.
func encodeOperationData(data string) string {
	if isValidHex(data) {
		return data
	}
	return asciiToHex(data)
}

// isValidHex checks if a string is valid hex (even length, all hex digits).
func isValidHex(s string) bool {
	if len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// toMCMSTimelockCalls converts local TimelockCall slice to mcms.TimelockCall slice for encoding.
func toMCMSTimelockCalls(calls []TimelockCall) []mcms.TimelockCall {
	result := make([]mcms.TimelockCall, len(calls))
	for i, c := range calls {
		result[i] = mcms.TimelockCall{
			TargetInstanceId: cantontypes.TEXT(c.TargetInstanceId),
			FunctionName:     cantontypes.TEXT(c.FunctionName),
			OperationData:    cantontypes.TEXT(c.OperationData),
		}
	}
	return result
}
