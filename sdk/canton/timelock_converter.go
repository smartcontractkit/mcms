package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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

	switch action {
	case types.TimelockActionSchedule:
		timelockFunctionName = "schedule_batch"
		operationDataEncoded = encodeScheduleBatchParams(calls, predecessorHex, saltHex, uint32(delay.Seconds()))

	case types.TimelockActionBypass:
		timelockFunctionName = "bypasser_execute_batch"
		operationDataEncoded = encodeBypasserExecuteParams(calls)

	case types.TimelockActionCancel:
		timelockFunctionName = "cancel_batch"
		// For cancel, the operationId is passed as the parameter
		operationDataEncoded = encodeCancelBatchParams(hex.EncodeToString(operationID[:]))

	default:
		return nil, common.Hash{}, fmt.Errorf("unsupported timelock action: %s", action)
	}

	// Build the Canton operation additional fields
	opAdditionalFields := AdditionalFields{
		TargetInstanceId: metadataFields.MultisigId, // Self-dispatch to MCMS
		FunctionName:     timelockFunctionName,
		OperationData:    operationDataEncoded,
		TargetCid:        mcmAddress, // MCMS contract ID
	}

	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to marshal operation additional fields: %w", err)
	}

	// Create the operation
	op := types.Operation{
		ChainSelector: bop.ChainSelector,
		Transaction: types.Transaction{
			To:               mcmAddress,
			Data:             nil, // Data is in AdditionalFields.OperationData
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

// encodeScheduleBatchParams encodes parameters for schedule_batch.
// Format: numCalls (1 byte) + calls + predecessor (text) + salt (text) + delay (4 bytes)
func encodeScheduleBatchParams(calls []TimelockCall, predecessor, salt string, delaySecs uint32) string {
	var encoded string

	// Encode calls list
	encoded += encodeUint8(uint8(len(calls)))
	for _, call := range calls {
		encoded += encodeTimelockCall(call)
	}

	// Encode predecessor and salt as length-prefixed text
	encoded += encodeText(predecessor)
	encoded += encodeText(salt)

	// Encode delay as 4-byte uint32
	encoded += encodeUint32(delaySecs)

	return encoded
}

// encodeBypasserExecuteParams encodes parameters for bypasser_execute_batch.
// Format: numCalls (1 byte) + calls
func encodeBypasserExecuteParams(calls []TimelockCall) string {
	var encoded string

	// Encode calls list
	encoded += encodeUint8(uint8(len(calls)))
	for _, call := range calls {
		encoded += encodeTimelockCall(call)
	}

	return encoded
}

// encodeCancelBatchParams encodes parameters for cancel_batch.
// Format: opId (text)
func encodeCancelBatchParams(opId string) string {
	return encodeText(opId)
}

// encodeTimelockCall encodes a single TimelockCall.
// Format: targetInstanceId (text) + functionName (text) + operationData (text)
func encodeTimelockCall(call TimelockCall) string {
	return encodeText(call.TargetInstanceId) + encodeText(call.FunctionName) + encodeText(call.OperationData)
}

// encodeText encodes a text string as length-prefixed bytes.
// Format: length (1 byte) + hex-encoded UTF-8 bytes
func encodeText(s string) string {
	hexStr := hex.EncodeToString([]byte(s))
	length := len(s)
	return encodeUint8(uint8(length)) + hexStr
}

// encodeUint8 encodes a uint8 as a 2-character hex string.
func encodeUint8(n uint8) string {
	return fmt.Sprintf("%02x", n)
}

// encodeUint32 encodes a uint32 as an 8-character hex string (big-endian).
func encodeUint32(n uint32) string {
	return fmt.Sprintf("%08x", n)
}
