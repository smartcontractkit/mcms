package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

var (
	// CallRevertedSelector is the 4-byte selector for the CallReverted error from MCMS 0x70de1b4b
	CallRevertedSelector = [4]byte{0x70, 0xde, 0x1b, 0x4b}
	// hexPattern matches "0x" followed by one or more hex characters
	hexPattern = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	// bytesArrayPattern matches "[[...]]" and captures the content between brackets
	bytesArrayPattern = regexp.MustCompile(`(?s)\[\[(.*)\]\]`)
	// customErrorPattern matches "custom error 0x<8-hex-chars>: <hex-data>"
	// Captures: group 1 = selector (8 hex chars), group 2 = data (hex chars with optional spaces)
	customErrorPattern = regexp.MustCompile(`custom error 0x([0-9a-fA-F]{8}):\s*([0-9a-fA-F\s]+)`)
)

const (
	selectorSize = 4
	revertPrefix = "revert:"
)

// CustomErrorData contains the error selector and its arguments separately.
type CustomErrorData struct {
	Selector [4]byte // 4-byte error selector
	Data     []byte  // Error arguments (ABI-encoded)
}

// MarshalJSON renders the selector/data as hex strings for readability.
func (c *CustomErrorData) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}

	payload := struct {
		Selector string `json:"selector"`
		Data     string `json:"data,omitempty"`
	}{
		Selector: "0x" + common.Bytes2Hex(c.Selector[:]),
	}

	if len(c.Data) > 0 {
		payload.Data = "0x" + common.Bytes2Hex(c.Data)
	}

	return json.Marshal(payload)
}

// UnmarshalJSON unmarshals JSON into CustomErrorData, handling hex string fields.
func (c *CustomErrorData) UnmarshalJSON(data []byte) error {
	if c == nil {
		return nil
	}

	var payload struct {
		Selector string `json:"selector"`
		Data     string `json:"data,omitempty"`
		Raw      string `json:"raw,omitempty"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if payload.Selector != "" {
		selectorBytes, err := hexutil.Decode(payload.Selector)
		if err == nil && len(selectorBytes) >= 4 {
			copy(c.Selector[:], selectorBytes[:4])
		}
	}

	if payload.Data != "" {
		dataBytes, err := hexutil.Decode(payload.Data)
		if err != nil {
			return err
		}
		c.Data = dataBytes
	}

	return nil
}

// Combined returns the full revert data (selector + data) as a byte slice.
func (c *CustomErrorData) Combined() []byte {
	if c == nil {
		return nil
	}

	return append(c.Selector[:], c.Data...)
}

// HexSelector returns the hex-encoded string representation of the selector (e.g., "0x70de1b4b").
func (c *CustomErrorData) HexSelector() string {
	if c == nil {
		return ""
	}

	return common.Bytes2Hex(c.Selector[:])
}

// ExecutionError represents an error that occurred during transaction execution.
// It includes the transaction data and revert reason for debugging and retry purposes.
type ExecutionError struct {
	// Transaction is the pre-packed transaction data that can be used to retry or simulate
	Transaction *gethtypes.Transaction
	// RawRevertReason contains the error selector and raw data from the contract
	// This allows for easy comparison of error selectors (e.g., to check if it's CallReverted)
	RawRevertReason *CustomErrorData
	// DecodedRevertReason is the human-readable decoded revert reason from the contract (e.g., "InsufficientSigners()")
	// This is only populated if the revert data could be decoded using contract ABIs
	DecodedRevertReason string
	// UnderlyingReason contains the raw revert reason returned by a nested contract call (plain string or hex)
	UnderlyingReason string
	// DecodedUnderlyingReason is the human-readable decoded underlying revert reason (if decoding succeeded)
	DecodedUnderlyingReason string
	// OriginalError is the original error from the contract binding
	OriginalError error `json:"-"`
}

func (e *ExecutionError) Error() string {
	if e.DecodedUnderlyingReason != "" {
		return fmt.Sprintf("execution failed: %v (underlying reason: %s)", e.OriginalError, e.DecodedUnderlyingReason)
	}
	if e.UnderlyingReason != "" {
		return fmt.Sprintf("execution failed: %v (underlying reason: %s)", e.OriginalError, e.UnderlyingReason)
	}
	if e.DecodedRevertReason != "" {
		return fmt.Sprintf("execution failed: %v (revert reason: %s)", e.OriginalError, e.DecodedRevertReason)
	}
	if e.RawRevertReason != nil && len(e.RawRevertReason.Combined()) > 0 {
		return fmt.Sprintf("execution failed: %v (raw revert data: %s)", e.OriginalError, common.Bytes2Hex(e.RawRevertReason.Combined()))
	}

	return fmt.Sprintf("execution failed: %v", e.OriginalError)
}

func (e *ExecutionError) Unwrap() error {
	return e.OriginalError
}

// MarshalJSON renders the execution error with a stringified OriginalError for portability.
func (e *ExecutionError) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}

	type alias ExecutionError
	copyAlias := alias(*e)
	copyAlias.OriginalError = nil

	payload := struct {
		*alias
		OriginalError string `json:"OriginalError,omitempty"`
	}{
		alias: &copyAlias,
	}

	if e.OriginalError != nil {
		payload.OriginalError = e.OriginalError.Error()
	}

	return json.Marshal(payload)
}

// UnmarshalJSON unmarshals JSON into ExecutionError, handling OriginalError as a string.
func (e *ExecutionError) UnmarshalJSON(data []byte) error {
	if e == nil {
		return nil
	}

	var jsonMap map[string]any
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}

	var originalErr error
	if origErrVal, ok := jsonMap["OriginalError"]; ok {
		if origErrStr, ok := origErrVal.(string); ok && origErrStr != "" {
			originalErr = errors.New(origErrStr)
		} else if origErrMap, ok := origErrVal.(map[string]any); ok && len(origErrMap) == 0 {
			originalErr = nil
		}
		delete(jsonMap, "OriginalError")
	}

	cleanData, err := json.Marshal(jsonMap)
	if err != nil {
		return err
	}

	type executionErrorAlias struct {
		Transaction             *gethtypes.Transaction `json:"Transaction"`
		RawRevertReason         *CustomErrorData       `json:"RawRevertReason"`
		DecodedRevertReason     string                 `json:"DecodedRevertReason"`
		UnderlyingReason        string                 `json:"UnderlyingReason"`
		DecodedUnderlyingReason string                 `json:"DecodedUnderlyingReason"`
	}

	var temp executionErrorAlias
	if err := json.Unmarshal(cleanData, &temp); err != nil {
		return err
	}

	e.Transaction = temp.Transaction
	e.RawRevertReason = temp.RawRevertReason
	e.DecodedRevertReason = temp.DecodedRevertReason
	e.UnderlyingReason = temp.UnderlyingReason
	e.DecodedUnderlyingReason = temp.DecodedUnderlyingReason
	if originalErr != nil {
		e.OriginalError = originalErr
	}

	return nil
}

// BuildExecutionError creates an ExecutionError from a contract execution error.
// It extracts revert reasons and handles special cases like RBACTimelock underlying transaction reverts.
// timelockAddr and timelockCallData are only needed for timelock execute cases (bypass or regular).
func BuildExecutionError(
	ctx context.Context,
	err error,
	txPreview *gethtypes.Transaction,
	opts *bind.TransactOpts,
	contractAddr common.Address,
	client ContractDeployBackend,
	timelockAddr common.Address,
	timelockCallData []byte,
) *ExecutionError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	execErr := &ExecutionError{
		Transaction:   txPreview,
		OriginalError: err,
	}

	// Extract both raw revert data and decoded revert reason from the error (best-effort)
	revertData := extractRevertReasonFromError(err)

	// If we have CustomErrorData directly, use it; otherwise construct from RawData
	if revertData.CustomError != nil {
		execErr.RawRevertReason = revertData.CustomError
	} else if len(revertData.RawData) > 0 {
		// Extract selector and data separately from raw data
		if len(revertData.RawData) >= selectorSize {
			var selector [selectorSize]byte
			copy(selector[:], revertData.RawData[:selectorSize])
			execErr.RawRevertReason = &CustomErrorData{
				Selector: selector,
				Data:     revertData.RawData[selectorSize:],
			}
		} else {
			execErr.RawRevertReason = &CustomErrorData{
				Selector: [selectorSize]byte{},
				Data:     revertData.RawData,
			}
		}
	}
	execErr.DecodedRevertReason = revertData.Decoded

	// Check if this is an RBACTimelock underlying transaction revert
	// This can appear in multiple ways:
	// 1. As a plain string in the error: "RBACTimelock: underlying transaction reverted"
	// 2. As a decoded custom error that contains this message
	// 3. As a CallReverted error (selector 0x70de1b4b) which wraps the timelock revert
	// In this case, we need to use CallContract to get the actual underlying revert reason.
	// If extraction fails, just leave UnderlyingReason empty
	rawRevertReasonHasCallRevertedSelector := execErr.RawRevertReason != nil && execErr.RawRevertReason.Selector == CallRevertedSelector
	rawDataHasCallRevertedSelector := len(revertData.RawData) >= selectorSize && string(revertData.RawData[:4]) == string(CallRevertedSelector[:])
	isCallReverted := strings.Contains(errStr, "CallReverted") ||
		strings.Contains(execErr.DecodedRevertReason, "CallReverted") ||
		rawRevertReasonHasCallRevertedSelector ||
		rawDataHasCallRevertedSelector

	if strings.Contains(errStr, "RBACTimelock: underlying transaction reverted") ||
		strings.Contains(execErr.DecodedRevertReason, "RBACTimelock: underlying transaction reverted") ||
		(isCallReverted && timelockAddr != (common.Address{}) && len(timelockCallData) > 0) {
		rawUnderlyingReason, decodedUnderlyingReason := getUnderlyingRevertReason(ctx, timelockAddr, timelockCallData, opts, client)
		execErr.UnderlyingReason = rawUnderlyingReason
		execErr.DecodedUnderlyingReason = decodedUnderlyingReason
	}

	return execErr
}

// extractHexEncodedRevertData extracts hex-encoded revert data (0x...) from an error string.
// Returns the extracted bytes if found, nil otherwise.
func extractHexEncodedRevertData(errStr string) []byte {
	hexStr := hexPattern.FindString(errStr)
	if hexStr == "" {
		return nil
	}

	if data := common.FromHex(hexStr); len(data) > 0 {
		return data
	}

	return nil
}

// extractBytesArrayRevertData extracts bytes array format revert data from an error string.
// Format: "[[...bytes...]]" where bytes are space-separated decimal values.
// Returns the extracted bytes if found, nil otherwise.
func extractBytesArrayRevertData(errStr string) []byte {
	matches := bytesArrayPattern.FindStringSubmatch(errStr)
	if len(matches) < 2 { //nolint
		return nil
	}

	bytesStr := matches[1] // First capture group contains the content between brackets
	data := parseBytesFromString(bytesStr)
	if len(data) > 0 {
		return data
	}

	return nil
}

// extractCustomErrorRevertData extracts revert data from the "custom error 0x...: <hex data>" format.
// Format: "execution reverted: custom error 0x70de1b4b: 00000000000000000000000000000000…00000000000000000000000000000000"
// Returns the error selector and data separately, or nil if not found.
func extractCustomErrorRevertData(errStr string) *CustomErrorData {
	matches := customErrorPattern.FindStringSubmatch(errStr)
	if len(matches) < 3 { //nolint
		return nil
	}

	// Extract selector (8 hex chars = 4 bytes) using shared hex extractor
	selectorBytes := extractHexEncodedRevertData("0x" + matches[1])
	if len(selectorBytes) != selectorSize {
		return nil
	}

	// Extract and clean data (remove spaces) before reusing the hex extractor
	dataHex := strings.ReplaceAll(matches[2], " ", "")
	data := extractHexEncodedRevertData("0x" + dataHex)
	if len(data) == 0 {
		return nil
	}

	var selector [4]byte
	copy(selector[:], selectorBytes)

	return &CustomErrorData{
		Selector: selector,
		Data:     data,
	}
}

// revertReasonData contains both the raw revert data and the decoded revert reason
type revertReasonData struct {
	RawData     []byte           // Raw hex-encoded revert data (full data including selector)
	Decoded     string           // Human-readable decoded revert reason
	CustomError *CustomErrorData // CustomErrorData if extracted directly (selector + data separately)
}

// extractRevertReasonFromError extracts both raw revert data and decoded revert reason from a bind error.
// Returns the raw data (hex-encoded) and the decoded reason (if decoding was successful).
func extractRevertReasonFromError(err error) revertReasonData {
	if err == nil {
		return revertReasonData{}
	}

	errStr := err.Error()

	// Check for plain string reverts first
	// (e.g., "execution reverted: revert: Ownable: caller is not the owner")
	if strings.Contains(errStr, revertPrefix) {
		// Try to extract the plain string revert reason
		// Format: "execution reverted: revert: <reason>" or "revert: <reason>"
		revertIdx := strings.Index(errStr, revertPrefix)
		if revertIdx != -1 {
			// Extract everything after "revert: "
			reason := strings.TrimSpace(errStr[revertIdx+len(revertPrefix):])
			if reason != "" {
				return revertReasonData{
					Decoded: reason,
				}
			}
		}
	}

	// Try to extract from "custom error 0x...: <hex data>" format
	if strings.Contains(errStr, "custom error") {
		customErr := extractCustomErrorRevertData(errStr)
		if customErr != nil {
			rawData := customErr.Combined()
			decoded := decodeRevertReasonFromCustomError(customErr)

			return revertReasonData{
				RawData:     rawData,
				Decoded:     decoded,
				CustomError: customErr, // Store CustomErrorData directly for easy selector access
			}
		}
	}

	// Try to extract revert data from error string using array, hex or bytesArray formats
	if strings.Contains(errStr, "CallReverted") {
		if rawData := extractBytesArrayRevertData(errStr); len(rawData) > 0 {
			return revertReasonData{
				RawData: rawData,
				Decoded: decodeRevertReason(rawData),
			}
		}
	}

	if rawData := extractHexEncodedRevertData(errStr); len(rawData) > 0 {
		return revertReasonData{
			RawData: rawData,
			Decoded: decodeRevertReason(rawData),
		}
	}

	if rawData := extractBytesArrayRevertData(errStr); len(rawData) > 0 {
		return revertReasonData{
			RawData: rawData,
			Decoded: decodeRevertReason(rawData),
		}
	}

	return revertReasonData{}
}

// parseBytesFromString parses a string representation of bytes like "8 195 121 160 ..."
func parseBytesFromString(s string) []byte {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	bytes := make([]byte, 0, len(parts))
	for _, part := range parts {
		if val, err := strconv.ParseUint(part, 10, 8); err == nil {
			bytes = append(bytes, byte(val))
		}
	}

	return bytes
}

// decodeRevertReasonFromCustomError decodes the revert reason using the error selector to find
// the matching error definition in the contract ABIs, then decodes the data part.
// Prioritizes selector matching for efficient error identification.
func decodeRevertReasonFromCustomError(customErr *CustomErrorData) string {
	if customErr == nil {
		return ""
	}

	selector := customErr.Selector[:]

	// Try to decode using contract ABIs (MCMS first, then Timelock)
	if decoded := tryDecodeWithContractABIs(selector, customErr.Data); decoded != "" {
		return decoded
	}

	// Fallback: try standard Solidity Error(string) or Panic(uint256) using go-ethereum's helper
	fullData := customErr.Combined()
	if reason, err := abi.UnpackRevert(fullData); err == nil {
		return reason
	}

	return ""
}

// decodeRevertReason decodes the revert reason from ABI-encoded data.
// First tries to decode using MCMS and RBACTimelock ABIs for custom errors.
// If that fails, falls back to Error(string) decoding.
// Returns empty string if all decoding fails (so we can return the original error).
func decodeRevertReason(data []byte) string {
	if len(data) < selectorSize {
		return ""
	}

	// Try to decode using contract ABIs (MCMS first, then Timelock)
	selector := data[:selectorSize]
	if decoded := tryDecodeWithContractABIs(selector, data[4:]); decoded != "" {
		return decoded
	}

	// Fallback to standard Solidity Error(string) or Panic(uint256) using go-ethereum's helper
	if reason, err := abi.UnpackRevert(data); err == nil {
		return reason
	}

	return ""
}

// tryDecodeWithContractABIs attempts to decode an error using MCMS and RBACTimelock ABIs.
// Returns the first successful decode, or empty string if all attempts fail.
func tryDecodeWithContractABIs(selector []byte, data []byte) string {
	if len(selector) != selectorSize {
		return ""
	}

	// Try MCMS ABI first
	if mcmsABI, err := bindings.ManyChainMultiSigMetaData.GetAbi(); err == nil && mcmsABI != nil {
		if decoded := decodeErrorBySelector(selector, data, mcmsABI); decoded != "" {
			return decoded
		}
	}

	// Try RBACTimelock ABI
	if timelockABI, err := bindings.RBACTimelockMetaData.GetAbi(); err == nil && timelockABI != nil {
		if decoded := decodeErrorBySelector(selector, data, timelockABI); decoded != "" {
			return decoded
		}
	}

	return ""
}

// decodeErrorBySelector finds an error in the ABI by matching the selector, then decodes the data part.
// The selector is used to identify the error, and only the data part (without selector) is decoded.
func decodeErrorBySelector(selector []byte, data []byte, contractABI *abi.ABI) string {
	if len(selector) != 4 || contractABI == nil {
		return ""
	}

	// Find the error in the ABI by matching the selector
	for name, errDef := range contractABI.Errors {
		if len(errDef.ID) < selectorSize {
			continue
		}
		errSelector := errDef.ID[:selectorSize]

		// Match the selector
		if string(errSelector) != string(selector) {
			continue
		}

		// Decode only the data part (without selector) using the error definition
		// The error definition expects the full data (selector + data), so we need to reconstruct it
		fullData := append(selector, data...)

		decoded, err := errDef.Unpack(fullData)
		if err != nil {
			if name == "CallReverted" && strings.Contains(err.Error(), "length insufficient") {
				return "CallReverted(truncated)"
			}

			continue
		}

		// Format the decoded error nicely
		var values []any
		if decodedSlice, ok := decoded.([]any); ok {
			values = decodedSlice
		} else {
			// If it's a single value or tuple, wrap it
			values = []any{decoded}
		}

		// Best-effort: if formatting fails, return empty string
		if formatted := formatDecodedError(name, values); formatted != "" {
			return formatted
		}
	}

	return ""
}

// formatDecodedError formats a decoded error into a readable string.
// Format as "ErrorName(arg1, arg2, ...)"
func formatDecodedError(errorName string, decodedValues []any) string {
	if errorName == "" {
		return ""
	}

	if len(decodedValues) == 0 {
		return errorName
	}

	var parts []string
	for _, val := range decodedValues {
		if val != nil {
			parts = append(parts, fmt.Sprintf("%v", val))
		}
	}
	if len(parts) > 0 {
		return fmt.Sprintf("%s(%s)", errorName, strings.Join(parts, ", "))
	}

	return errorName
}

// getUnderlyingRevertReason extracts both the raw and decoded underlying revert reasons when MCMS calls
// RBACTimelock's bypasserExecuteBatch or executeBatch and one of the underlying transactions reverts.
// This function:
// 1. Extracts the underlying call from the timelock execute calls array
// 2. Simulates the underlying call using the timelock address as From (since that's msg.sender)
// 3. Returns the raw revert data/message and the decoded revert reason (if decoding succeeds)
func getUnderlyingRevertReason(
	ctx context.Context,
	timelockAddr common.Address,
	timelockCallData []byte,
	opts *bind.TransactOpts,
	client ContractDeployBackend,
) (string, string) {
	if timelockAddr == (common.Address{}) || len(timelockCallData) == 0 || client == nil || opts == nil {
		return "", ""
	}

	// Extract the underlying call from timelock execute calls array
	underlyingCall := extractUnderlyingCall(timelockCallData)
	if underlyingCall == nil {
		return "", ""
	}

	// Simulate the underlying transaction using CallContract (best-effort)
	// Use timelock address as From since that's what msg.sender will be when RBACTimelock executes
	_, err := client.CallContract(ctx, ethereum.CallMsg{
		From:  timelockAddr,
		To:    &underlyingCall.Target,
		Value: underlyingCall.Value,
		Data:  underlyingCall.Data,
		Gas:   opts.GasLimit,
	}, nil)

	if err == nil {
		return "", ""
	}

	errStr := err.Error()

	// Check if error contains revert data
	if !strings.Contains(errStr, "execution reverted") && !strings.Contains(errStr, "revert") {
		return "", ""
	}

	var rawReason string
	var decodedReason string

	// Try to extract revert data from error string (best-effort)
	// First try direct extraction, then fall back to full error parsing
	if revertDataBytes := extractRevertDataFromCallError(err); len(revertDataBytes) > 0 {
		rawReason = "0x" + common.Bytes2Hex(revertDataBytes)
		if reason := decodeRevertReason(revertDataBytes); reason != "" {
			decodedReason = reason
		}
	}

	// Fall back to full error parsing
	revertData := extractRevertReasonFromError(err)
	if len(revertData.RawData) > 0 && rawReason == "" {
		rawReason = "0x" + common.Bytes2Hex(revertData.RawData)
	}
	if decodedReason == "" && revertData.Decoded != "" {
		decodedReason = revertData.Decoded
	}
	if rawReason == "" && revertData.Decoded != "" {
		rawReason = revertData.Decoded
	}
	if rawReason == "" {
		if idx := strings.Index(errStr, revertPrefix); idx != -1 {
			rawReason = strings.TrimSpace(errStr[idx+len(revertPrefix):])
		}
	}
	if rawReason == "" {
		rawReason = errStr
	}

	return rawReason, decodedReason
}

// extractRevertDataFromCallError extracts revert data from a CallContract error.
// This is a best-effort extraction as the format may vary.
func extractRevertDataFromCallError(err error) []byte {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Try to find hex-encoded revert data (0x...)
	if data := extractHexEncodedRevertData(errStr); len(data) > 0 {
		return data
	}

	// Try to extract bytes array format
	if data := extractBytesArrayRevertData(errStr); len(data) > 0 {
		return data
	}

	return nil
}

// extractUnderlyingCall extracts the first underlying call from timelock execute calldata.
// Works with both bypasserExecuteBatch(calls RBACTimelockCall[]) and
// executeBatch(calls RBACTimelockCall[], predecessor, salt).
// This is a modular helper that can be reused for both bypass and regular timelock execute cases.
func extractUnderlyingCall(timelockCallData []byte) *bindings.RBACTimelockCall {
	if len(timelockCallData) < selectorSize {
		return nil
	}

	// Get the RBACTimelock ABI (best-effort)
	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	if err != nil || timelockABI == nil {
		return nil
	}

	// Try bypasserExecuteBatch first
	if method, exists := timelockABI.Methods["bypasserExecuteBatch"]; exists {
		if len(method.ID) >= 4 && string(timelockCallData[:4]) == string(method.ID[:4]) {
			if result := extractCallFromMethod(timelockCallData, &method); result != nil {
				return result
			}
		}
	}

	// Try executeBatch
	if method, exists := timelockABI.Methods["executeBatch"]; exists {
		if len(method.ID) >= 4 && string(timelockCallData[:4]) == string(method.ID[:4]) {
			return extractCallFromMethod(timelockCallData, &method)
		}
	}

	return nil
}

// extractCallFromMethod extracts the first call from a timelock execute method's calldata.
// Both bypasserExecuteBatch and executeBatch have calls as their first parameter.
// Best-effort: returns nil if extraction fails.
func extractCallFromMethod(callData []byte, method *abi.Method) *bindings.RBACTimelockCall {
	firstCall := firstTimelockCallArgument(callData, method)
	if firstCall == nil {
		return nil
	}

	if call, ok := firstCall.(bindings.RBACTimelockCall); ok {
		return &call
	}

	return extractCallFields(firstCall)
}

// firstTimelockCallArgument returns the first element of the timelock calls slice extracted from calldata.
func firstTimelockCallArgument(callData []byte, method *abi.Method) any {
	if method == nil || len(callData) < selectorSize {
		return nil
	}

	args, err := method.Inputs.UnpackValues(callData[selectorSize:])
	if err != nil || len(args) == 0 {
		return nil
	}

	argValue := reflect.ValueOf(args[0])
	if !argValue.IsValid() || argValue.Kind() != reflect.Slice || argValue.Len() == 0 {
		return nil
	}

	return argValue.Index(0).Interface()
}

// extractCallFields extracts target, value, and data from a call structure.
// Handles the formats that go-ethereum's ABI unpacking may return:
// - Direct struct type (most common)
// - Anonymous struct with same fields (from UnpackValues)
// - Map format (from UnpackIntoMap or named tuples)
// - Tuple slice format (positional tuples with unnamed fields)
func extractCallFields(call any) *bindings.RBACTimelockCall {
	if callStruct, ok := call.(bindings.RBACTimelockCall); ok {
		return &callStruct
	}

	if callStruct := extractCallFromStruct(call); callStruct != nil {
		return callStruct
	}

	if callStruct := extractCallFromMap(call); callStruct != nil {
		return callStruct
	}

	return extractCallFromTuple(call)
}

func extractCallFromStruct(call any) *bindings.RBACTimelockCall {
	callValue := reflect.ValueOf(call)
	if !callValue.IsValid() || callValue.Kind() != reflect.Struct {
		return nil
	}

	target := addressFromField(callValue, "Target")
	if target == (common.Address{}) {
		return nil
	}

	value := bigIntFromField(callValue, "Value")
	data := bytesFromField(callValue, "Data")

	return newTimelockCall(target, value, data)
}

func extractCallFromMap(call any) *bindings.RBACTimelockCall {
	callMap, ok := call.(map[string]any)
	if !ok {
		return nil
	}

	target := addressFromMap(callMap, "target", "Target")
	if target == (common.Address{}) {
		return nil
	}

	value := bigIntFromMap(callMap, "value", "Value")
	data := bytesFromMap(callMap, "data", "Data")

	return newTimelockCall(target, value, data)
}

func extractCallFromTuple(call any) *bindings.RBACTimelockCall {
	callSlice, ok := call.([]any)
	if !ok || len(callSlice) < 3 {
		return nil
	}

	target, _ := callSlice[0].(common.Address)
	if target == (common.Address{}) {
		return nil
	}

	value, _ := callSlice[1].(*big.Int)
	data := bytesFromValue(callSlice[2])

	return newTimelockCall(target, value, data)
}

func addressFromField(value reflect.Value, fieldName string) common.Address {
	field := value.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return common.Address{}
	}
	addr, _ := field.Interface().(common.Address)

	return addr
}

func bigIntFromField(value reflect.Value, fieldName string) *big.Int {
	field := value.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}
	bigVal, _ := field.Interface().(*big.Int)

	return bigVal
}

func bytesFromField(value reflect.Value, fieldName string) []byte {
	field := value.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}

	return bytesFromValue(field.Interface())
}

func addressFromMap(m map[string]any, keys ...string) common.Address {
	for _, key := range keys {
		if addr, ok := m[key].(common.Address); ok {
			return addr
		}
	}

	return common.Address{}
}

func bigIntFromMap(m map[string]any, keys ...string) *big.Int {
	for _, key := range keys {
		if val, ok := m[key].(*big.Int); ok {
			return val
		}
	}

	return nil
}

func bytesFromMap(m map[string]any, keys ...string) []byte {
	for _, key := range keys {
		if data := bytesFromValue(m[key]); len(data) > 0 {
			return data
		}
	}

	return nil
}

func bytesFromValue(value any) []byte {
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return data
	}

	copied := make([]byte, len(data))
	copy(copied, data)

	return copied
}

func newTimelockCall(target common.Address, value *big.Int, data []byte) *bindings.RBACTimelockCall {
	if target == (common.Address{}) {
		return nil
	}

	return &bindings.RBACTimelockCall{
		Target: target,
		Value:  value,
		Data:   data,
	}
}
