package evm

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/evm/mocks"
)

func TestExecutionError_Error(t *testing.T) {
	tests := []struct {
		name                string
		execErr             *ExecutionError
		expectedContains    []string // substrings that should be in the error message
		expectedNotContains []string // substrings that should NOT be in the error message
	}{
		{
			name: "with underlying reason - highest priority",
			execErr: &ExecutionError{
				OriginalError:       errors.New("original error"),
				UnderlyingReason:    "underlying revert reason",
				DecodedRevertReason: "decoded reason",
				RawRevertReason: &CustomErrorData{
					Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
					Data:     []byte{0xaa, 0xbb, 0xcc},
				},
			},
			expectedContains: []string{
				"execution failed",
				"original error",
				"underlying reason: underlying revert reason",
			},
			expectedNotContains: []string{
				"revert reason: decoded reason", // should not show decoded reason
				"raw revert data",               // should not show raw data
			},
		},
		{
			name: "with decoded revert reason - second priority",
			execErr: &ExecutionError{
				OriginalError:       errors.New("original error"),
				DecodedRevertReason: "OutOfBoundsGroup",
				RawRevertReason: &CustomErrorData{
					Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
					Data:     []byte{0xaa, 0xbb, 0xcc},
				},
			},
			expectedContains: []string{
				"execution failed",
				"original error",
				"revert reason: OutOfBoundsGroup",
			},
			expectedNotContains: []string{
				"underlying reason",
				"raw revert data",
			},
		},
		{
			name: "with raw revert reason - third priority",
			execErr: &ExecutionError{
				OriginalError: errors.New("original error"),
				RawRevertReason: &CustomErrorData{
					Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
					Data:     []byte{0xaa, 0xbb, 0xcc},
				},
			},
			expectedContains: []string{
				"execution failed",
				"original error",
				"raw revert data",
				"12345678aabbcc", // hex representation of selector + data
			},
			expectedNotContains: []string{
				"underlying reason",
				"revert reason",
			},
		},
		{
			name: "with empty raw revert reason - selector only still shows raw data",
			execErr: &ExecutionError{
				OriginalError: errors.New("original error"),
				RawRevertReason: &CustomErrorData{
					Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
					Data:     []byte{}, // empty data, but selector is present
				},
			},
			expectedContains: []string{
				"execution failed",
				"original error",
				"raw revert data",
				"12345678", // selector hex
			},
			expectedNotContains: []string{
				"underlying reason",
				"revert reason:",
			},
		},
		{
			name: "with nil raw revert reason - should not show raw data",
			execErr: &ExecutionError{
				OriginalError:   errors.New("original error"),
				RawRevertReason: nil,
			},
			expectedContains: []string{
				"execution failed",
				"original error",
			},
			expectedNotContains: []string{
				"underlying reason",
				"revert reason",
				"raw revert data",
			},
		},
		{
			name: "only original error - fallback case",
			execErr: &ExecutionError{
				OriginalError: errors.New("original error"),
			},
			expectedContains: []string{
				"execution failed",
				"original error",
			},
			expectedNotContains: []string{
				"underlying reason",
				"revert reason",
				"raw revert data",
			},
		},
		{
			name: "with CallReverted selector",
			execErr: &ExecutionError{
				OriginalError: errors.New("original error"),
				RawRevertReason: &CustomErrorData{
					Selector: CallRevertedSelector,
					Data:     []byte{0xaa, 0xbb, 0xcc},
				},
			},
			expectedContains: []string{
				"execution failed",
				"original error",
				"raw revert data",
				"70de1b4b", // CallReverted selector
			},
		},
		{
			name: "with decoded reason but no underlying reason",
			execErr: &ExecutionError{
				OriginalError:       errors.New("contract call failed"),
				DecodedRevertReason: "InsufficientSigners",
			},
			expectedContains: []string{
				"execution failed",
				"contract call failed",
				"revert reason: InsufficientSigners",
			},
			expectedNotContains: []string{
				"underlying reason",
				"raw revert data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.execErr.Error()

			// Check that all expected substrings are present
			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected, "Error message should contain: %s", expected)
			}

			// Check that unexpected substrings are not present
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, result, notExpected, "Error message should NOT contain: %s", notExpected)
			}
		})
	}
}

func TestExecutionError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	execErr := &ExecutionError{
		OriginalError: originalErr,
	}

	unwrapped := execErr.Unwrap()
	require.Equal(t, originalErr, unwrapped, "Unwrap should return the original error")
}

func TestCustomErrorData(t *testing.T) {
	tests := []struct {
		name             string
		customErr        *CustomErrorData
		expectedHex      string
		expectedCombined []byte
	}{
		{
			name:             "nil CustomErrorData",
			customErr:        nil,
			expectedHex:      "",
			expectedCombined: nil,
		},
		{
			name: "CallReverted selector",
			customErr: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     []byte{0xaa, 0xbb, 0xcc},
			},
			expectedHex:      "70de1b4b",
			expectedCombined: []byte{0x70, 0xde, 0x1b, 0x4b, 0xaa, 0xbb, 0xcc},
		},
		{
			name: "custom selector with empty data",
			customErr: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     []byte{},
			},
			expectedHex:      "12345678",
			expectedCombined: []byte{0x12, 0x34, 0x56, 0x78},
		},
		{
			name: "custom selector with data",
			customErr: &CustomErrorData{
				Selector: [4]byte{0xAB, 0xCD, 0xEF, 0x01},
				Data:     []byte{0x11, 0x22, 0x33, 0x44},
			},
			expectedHex:      "abcdef01",
			expectedCombined: []byte{0xAB, 0xCD, 0xEF, 0x01, 0x11, 0x22, 0x33, 0x44},
		},
		{
			name: "zero selector",
			customErr: &CustomErrorData{
				Selector: [4]byte{0x00, 0x00, 0x00, 0x00},
				Data:     []byte{0xFF},
			},
			expectedHex:      "00000000",
			expectedCombined: []byte{0x00, 0x00, 0x00, 0x00, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test HexSelector()
			hexSelector := tt.customErr.HexSelector()
			assert.Equal(t, tt.expectedHex, hexSelector, "HexSelector() mismatch")

			// Test Combined()
			combined := tt.customErr.Combined()
			assert.Equal(t, tt.expectedCombined, combined, "Combined() mismatch")
		})
	}
}

func TestExtractHexEncodedRevertData(t *testing.T) {
	tests := []struct {
		name        string
		errStr      string
		expected    []byte
		shouldMatch bool
	}{
		{
			name:        "valid hex string",
			errStr:      "execution reverted: 0x1234567890abcdef",
			expected:    common.FromHex("0x1234567890abcdef"),
			shouldMatch: true,
		},
		{
			name:        "mixed case hex",
			errStr:      "0xAbCdEf123456",
			expected:    common.FromHex("0xAbCdEf123456"),
			shouldMatch: true,
		},
		{
			name:        "hex stops at non-hex character",
			errStr:      "0x123456xyz789",
			expected:    common.FromHex("0x123456"),
			shouldMatch: true,
		},
		{
			name:        "multiple hex strings - matches first",
			errStr:      "0x123456 0xabcdef",
			expected:    common.FromHex("0x123456"),
			shouldMatch: true,
		},
		{
			name:        "empty string",
			errStr:      "",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "no hex pattern",
			errStr:      "just some text without hex",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "only 0x prefix",
			errStr:      "0x",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "0x followed by non-hex",
			errStr:      "0xghijkl",
			expected:    nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHexEncodedRevertData(tt.errStr)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil for input: %q", tt.errStr)
				return
			}

			require.NotNil(t, result, "Expected non-nil result for input: %q", tt.errStr)
			assert.Equal(t, tt.expected, result, "Extracted hex data mismatch for input: %q", tt.errStr)
		})
	}
}

func TestExtractBytesArrayRevertData(t *testing.T) {
	tests := []struct {
		name        string
		errStr      string
		expected    []byte
		shouldMatch bool
	}{
		{
			name:        "valid bytes array",
			errStr:      "[[8 195 121 160]]",
			expected:    []byte{8, 195, 121, 160},
			shouldMatch: true,
		},
		{
			name:        "bytes array with whitespace variations",
			errStr:      "[[8  195\t121\n160]]",
			expected:    []byte{8, 195, 121, 160},
			shouldMatch: true,
		},
		{
			name:        "bytes array with invalid values - skips invalid",
			errStr:      "[[8 abc 195 256 -1 121]]",
			expected:    []byte{8, 195, 121},
			shouldMatch: true,
		},
		{
			name:        "empty brackets",
			errStr:      "[[]]",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "all invalid values",
			errStr:      "[[abc xyz 999]]",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "no brackets",
			errStr:      "8 195 121 160",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "incomplete brackets",
			errStr:      "[[8 195 121",
			expected:    nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBytesArrayRevertData(tt.errStr)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil for input: %q", tt.errStr)
				return
			}

			require.NotNil(t, result, "Expected non-nil result for input: %q", tt.errStr)
			assert.Equal(t, tt.expected, result, "Extracted bytes mismatch for input: %q", tt.errStr)
		})
	}
}

func TestExtractCustomErrorRevertData(t *testing.T) {
	tests := []struct {
		name        string
		errStr      string
		expected    *CustomErrorData
		shouldMatch bool
	}{
		{
			name:   "valid custom error",
			errStr: "execution reverted: custom error 0x70de1b4b: aabbccdd",
			expected: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     common.FromHex("0xaabbccdd"),
			},
			shouldMatch: true,
		},
		{
			name:   "mixed case hex with spaces in data",
			errStr: "execution reverted: custom error 0xAbCd1234: aa bb cc dd",
			expected: &CustomErrorData{
				Selector: [4]byte{0xAB, 0xCD, 0x12, 0x34},
				Data:     common.FromHex("0xaabbccdd"),
			},
			shouldMatch: true,
		},
		{
			name:   "with ellipsis truncation",
			errStr: "execution reverted: custom error 0x70de1b4b: aabbcc…ddeeff",
			expected: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     common.FromHex("0xaabbcc"),
			},
			shouldMatch: true,
		},
		{
			name:   "stops at non-hex character",
			errStr: "execution reverted: custom error 0x12345678: aabbccddxyz",
			expected: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     common.FromHex("0xaabbccdd"),
			},
			shouldMatch: true,
		},
		{
			name:        "empty string",
			errStr:      "",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "no custom error pattern",
			errStr:      "execution reverted: some other error",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "invalid selector length",
			errStr:      "execution reverted: custom error 0x123: aabbccdd",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "no data after colon",
			errStr:      "execution reverted: custom error 0x12345678:",
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "invalid hex in data",
			errStr:      "execution reverted: custom error 0x12345678: xyz",
			expected:    nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCustomErrorRevertData(tt.errStr)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil for input: %q", tt.errStr)
				return
			}

			require.NotNil(t, result, "Expected non-nil result for input: %q", tt.errStr)
			assert.Equal(t, tt.expected.Selector, result.Selector, "Selector mismatch for input: %q", tt.errStr)
			assert.Equal(t, tt.expected.Data, result.Data, "Data mismatch for input: %q", tt.errStr)
		})
	}
}

func TestParseBytesFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "valid bytes string",
			input:    "8 195 121 160",
			expected: []byte{8, 195, 121, 160},
		},
		{
			name:     "with whitespace variations",
			input:    "  8\t195\n121   160  ",
			expected: []byte{8, 195, 121, 160},
		},
		{
			name:     "edge values",
			input:    "0 128 255",
			expected: []byte{0, 128, 255},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: nil,
		},
		{
			name:     "invalid values skipped",
			input:    "8 abc 121 256 -1 160",
			expected: []byte{8, 121, 160},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBytesFromString(tt.input)
			assert.Equal(t, tt.expected, result, "Parsed bytes mismatch for input: %q", tt.input)
		})
	}
}

func TestExtractRevertReasonFromError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedRawData  []byte
		hasCustomError   bool
		expectedSelector [4]byte
		shouldHaveData   bool
		priority         string // "custom", "callreverted", "hex", "bytesarray", "none"
	}{
		{
			name:            "nil error",
			err:             nil,
			expectedRawData: nil,
			hasCustomError:  false,
			shouldHaveData:  false,
			priority:        "none",
		},
		{
			name:             "custom error format - highest priority",
			err:              errors.New("execution reverted: custom error 0x70de1b4b: aabbccdd"),
			expectedRawData:  append(CallRevertedSelector[:], common.FromHex("0xaabbccdd")...),
			hasCustomError:   true,
			expectedSelector: CallRevertedSelector,
			shouldHaveData:   true,
			priority:         "custom",
		},
		{
			name:            "CallReverted format with bytes array",
			err:             errors.New("contract error: error -`CallReverted` args [[8 195 121 160]]"),
			expectedRawData: []byte{8, 195, 121, 160},
			hasCustomError:  false,
			shouldHaveData:  true,
			priority:        "callreverted",
		},
		{
			name:            "hex-encoded format",
			err:             errors.New("execution reverted: 0x1234567890abcdef"),
			expectedRawData: common.FromHex("0x1234567890abcdef"),
			hasCustomError:  false,
			shouldHaveData:  true,
			priority:        "hex",
		},
		{
			name:            "bytes array format (without CallReverted)",
			err:             errors.New("error with bytes [[8 195 121 160]]"),
			expectedRawData: []byte{8, 195, 121, 160},
			hasCustomError:  false,
			shouldHaveData:  true,
			priority:        "bytesarray",
		},
		{
			name:            "plain error without revert data",
			err:             errors.New("some generic error"),
			expectedRawData: nil,
			hasCustomError:  false,
			shouldHaveData:  false,
			priority:        "none",
		},
		{
			name:            "invalid formats fall through",
			err:             errors.New("CallReverted but no bytes array"),
			expectedRawData: nil,
			hasCustomError:  false,
			shouldHaveData:  false,
			priority:        "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRevertReasonFromError(tt.err)

			if !tt.shouldHaveData {
				assert.Nil(t, result.RawData, "Expected nil RawData for error: %v", tt.err)
				assert.Nil(t, result.CustomError, "Expected nil CustomError for error: %v", tt.err)
				return
			}

			require.NotNil(t, result.RawData, "Expected non-nil RawData for error: %v", tt.err)
			assert.Equal(t, tt.expectedRawData, result.RawData, "RawData mismatch for error: %v", tt.err)

			if tt.hasCustomError {
				require.NotNil(t, result.CustomError, "Expected non-nil CustomError for error: %v", tt.err)
				assert.Equal(t, tt.expectedSelector, result.CustomError.Selector, "Selector mismatch for error: %v", tt.err)
				// Verify Combined() matches RawData
				assert.Equal(t, result.RawData, result.CustomError.Combined(), "CustomError.Combined() should match RawData")
			} else {
				assert.Nil(t, result.CustomError, "Expected nil CustomError for error: %v", tt.err)
			}
		})
	}
}

func TestDecodeRevertReasonFromCustomError(t *testing.T) {
	// Error(string) selector: keccak256("Error(string)")[:4] = 0x08c379a0
	errorStringSelector := [4]byte{0x08, 0xc3, 0x79, 0xa0}

	tests := []struct {
		name           string
		customErr      *CustomErrorData
		expectedResult string // Expected decoded result (empty string means decoding failed)
		shouldDecode   bool   // Whether we expect a non-empty decoded result
		description    string // Description of what we're testing
	}{
		{
			name:           "nil CustomErrorData",
			customErr:      nil,
			expectedResult: "",
			shouldDecode:   false,
			description:    "Should return empty string for nil input",
		},
		{
			name: "CallReverted with empty data",
			customErr: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     []byte{},
			},
			expectedResult: "",    // May decode to "CallReverted([])" or empty depending on ABI
			shouldDecode:   false, // Empty data might not decode properly
			description:    "CallReverted with empty data",
		},
		{
			name: "CallReverted with valid data",
			customErr: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // Some valid data
			},
			expectedResult: "CallReverted(truncated)", // Returns truncated marker when data is insufficient
			shouldDecode:   true,
			description:    "CallReverted with insufficient data returns truncated marker",
		},
		{
			name: "Error(string) with valid string data",
			customErr: &CustomErrorData{
				Selector: errorStringSelector,
				Data:     encodeErrorString("test error message"),
			},
			expectedResult: "test error message", // Now works with proper ABI encoding
			shouldDecode:   true,
			description:    "Error(string) decoding with proper ABI encoding",
		},
		{
			name: "Error(string) with empty string",
			customErr: &CustomErrorData{
				Selector: errorStringSelector,
				Data:     encodeErrorString(""),
			},
			expectedResult: "",
			shouldDecode:   false, // May not decode with manual encoding
			description:    "Error(string) with empty string",
		},
		{
			name: "unknown selector",
			customErr: &CustomErrorData{
				Selector: [4]byte{0xFF, 0xFF, 0xFF, 0xFF},
				Data:     []byte{0x12, 0x34, 0x56, 0x78},
			},
			expectedResult: "",
			shouldDecode:   false,
			description:    "Unknown selector should return empty string",
		},
		{
			name: "Error(string) with malformed data",
			customErr: &CustomErrorData{
				Selector: errorStringSelector,
				Data:     []byte{0x00, 0x00, 0x00, 0x00}, // Too short for proper encoding
			},
			expectedResult: "",
			shouldDecode:   false,
			description:    "Error(string) with malformed/incomplete data should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeRevertReasonFromCustomError(tt.customErr)

			if !tt.shouldDecode {
				assert.Empty(t, result, "Expected empty result for: %s. Got: %q", tt.description, result)
			} else {
				assert.NotEmpty(t, result, "Expected non-empty result for: %s", tt.description)
				if tt.expectedResult != "" {
					assert.Equal(t, tt.expectedResult, result, "Decoded result mismatch for: %s", tt.description)
				}
			}
		})
	}
}

func TestFormatDecodedError(t *testing.T) {
	tests := []struct {
		name           string
		errorName      string
		decodedValues  []any
		expectedResult string
	}{
		{
			name:           "empty error name",
			errorName:      "",
			decodedValues:  []any{"arg1", "arg2"},
			expectedResult: "",
		},
		{
			name:           "no arguments",
			errorName:      "InsufficientSigners",
			decodedValues:  []any{},
			expectedResult: "InsufficientSigners",
		},
		{
			name:           "single argument",
			errorName:      "OutOfBoundsGroup",
			decodedValues:  []any{uint8(255)},
			expectedResult: "OutOfBoundsGroup(255)",
		},
		{
			name:           "multiple arguments",
			errorName:      "CustomError",
			decodedValues:  []any{"arg1", 123, true},
			expectedResult: "CustomError(arg1, 123, true)",
		},
		{
			name:           "with nil values - nil skipped",
			errorName:      "ErrorWithNil",
			decodedValues:  []any{"arg1", nil, "arg2"},
			expectedResult: "ErrorWithNil(arg1, arg2)",
		},
		{
			name:           "all nil values",
			errorName:      "ErrorWithAllNil",
			decodedValues:  []any{nil, nil},
			expectedResult: "ErrorWithAllNil",
		},
		{
			name:           "address argument",
			errorName:      "AddressError",
			decodedValues:  []any{common.HexToAddress("0x1234567890123456789012345678901234567890")},
			expectedResult: "AddressError(0x1234567890123456789012345678901234567890)",
		},
		{
			name:           "big int argument",
			errorName:      "BigIntError",
			decodedValues:  []any{big.NewInt(1000)},
			expectedResult: "BigIntError(1000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDecodedError(tt.errorName, tt.decodedValues)
			assert.Equal(t, tt.expectedResult, result, "Format mismatch for errorName=%q, values=%v", tt.errorName, tt.decodedValues)
		})
	}
}

func TestExtractRevertDataFromCallError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expected    []byte
		shouldMatch bool
	}{
		{
			name:        "nil error",
			err:         nil,
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "error with hex-encoded revert data",
			err:         errors.New("execution reverted: 0x1234567890abcdef"),
			expected:    common.FromHex("0x1234567890abcdef"),
			shouldMatch: true,
		},
		{
			name:        "error with bytes array format",
			err:         errors.New("execution reverted: CallReverted args [[8 195 121 160]]"),
			expected:    []byte{8, 195, 121, 160},
			shouldMatch: true,
		},
		{
			name:        "error without revert data",
			err:         errors.New("some generic error"),
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "error with both formats - hex takes priority",
			err:         errors.New("execution reverted: 0x12345678 and bytes [[8 195 121]]"),
			expected:    common.FromHex("0x12345678"),
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRevertDataFromCallError(tt.err)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil for error: %v", tt.err)
				return
			}

			require.NotNil(t, result, "Expected non-nil result for error: %v", tt.err)
			assert.Equal(t, tt.expected, result, "Extracted revert data mismatch for error: %v", tt.err)
		})
	}
}

func TestExtractCallFields(t *testing.T) {
	testTarget := common.HexToAddress("0x1234567890123456789012345678901234567890")

	tests := []struct {
		name        string
		input       any
		want        *bindings.RBACTimelockCall
		shouldMatch bool
	}{
		{
			name: "map_with_lowercase_keys_and_uint8_data",
			input: map[string]any{
				"target": testTarget,
				"value":  big.NewInt(42),
				"data":   []uint8{0xaa, 0xbb},
			},
			want: &bindings.RBACTimelockCall{
				Target: testTarget,
				Value:  big.NewInt(42),
				Data:   []byte{0xaa, 0xbb},
			},
			shouldMatch: true,
		},
		{
			name: "map_with_uppercase_keys",
			input: map[string]any{
				"Target": testTarget,
				"Value":  (*big.Int)(nil),
				"Data":   []byte{0x01, 0x02, 0x03},
			},
			want: &bindings.RBACTimelockCall{
				Target: testTarget,
				Value:  nil,
				Data:   []byte{0x01, 0x02, 0x03},
			},
			shouldMatch: true,
		},
		{
			name: "map_with_uppercase_uint8_data",
			input: map[string]any{
				"Target": testTarget,
				"Value":  big.NewInt(7),
				"Data":   []uint8{0xde, 0xad},
			},
			want: &bindings.RBACTimelockCall{
				Target: testTarget,
				Value:  big.NewInt(7),
				Data:   []byte{0xde, 0xad},
			},
			shouldMatch: true,
		},
		{
			name: "tuple_slice_format",
			input: []any{
				testTarget,
				big.NewInt(1000),
				[]uint8{0xcc, 0xdd},
			},
			want: &bindings.RBACTimelockCall{
				Target: testTarget,
				Value:  big.NewInt(1000),
				Data:   []byte{0xcc, 0xdd},
			},
			shouldMatch: true,
		},
		{
			name: "map_with_zero_target_returns_nil",
			input: map[string]any{
				"target": common.Address{},
				"value":  big.NewInt(1),
				"data":   []byte{0x01},
			},
			want:        nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCallFields(tt.input)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil result for test: %s", tt.name)
				return
			}

			require.NotNil(t, result, "Expected non-nil result for test: %s", tt.name)
			assert.Equal(t, tt.want.Target, result.Target, "Target mismatch for test: %s", tt.name)
			if tt.want.Value == nil {
				assert.Nil(t, result.Value, "Expected nil value for test: %s", tt.name)
			} else {
				require.NotNil(t, result.Value, "Expected non-nil value for test: %s", tt.name)
				assert.Equal(t, tt.want.Value, result.Value, "Value mismatch for test: %s", tt.name)
			}
			assert.Equal(t, tt.want.Data, result.Data, "Data mismatch for test: %s", tt.name)
		})
	}
}

func TestExtractCallFromMethod(t *testing.T) {
	// Get the real timelock ABI for testing
	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, "Failed to get RBACTimelock ABI")

	bypassMethod, exists := timelockABI.Methods["bypasserExecuteBatch"]
	require.True(t, exists, "bypasserExecuteBatch method not found")

	// Create test call data
	testTarget := common.HexToAddress("0x1234567890123456789012345678901234567890")
	testValue := big.NewInt(1000)
	testData := []byte{0x12, 0x34, 0x56, 0x78}
	testCall := bindings.RBACTimelockCall{
		Target: testTarget,
		Value:  testValue,
		Data:   testData,
	}

	tests := []struct {
		name        string
		callData    []byte
		method      *abi.Method
		expected    *bindings.RBACTimelockCall
		shouldMatch bool
	}{
		{
			name:        "nil method",
			callData:    []byte{0x12, 0x34, 0x56, 0x78},
			method:      nil,
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "callData too short",
			callData:    []byte{0x12, 0x34}, // Less than 4 bytes
			method:      &bypassMethod,
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "valid bypasserExecuteBatch with single call",
			callData:    packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{testCall}),
			method:      &bypassMethod,
			expected:    &testCall,
			shouldMatch: true,
		},
		{
			name: "valid bypasserExecuteBatch with multiple calls - extracts first",
			callData: packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{testCall, {
				Target: common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"),
				Value:  big.NewInt(2000),
				Data:   []byte{0xab, 0xcd, 0xef},
			}}),
			method:      &bypassMethod,
			expected:    &testCall,
			shouldMatch: true,
		},
		{
			name:        "invalid calldata - unpack fails",
			callData:    append(bypassMethod.ID, make([]byte, 100)...), // Valid selector but invalid data
			method:      &bypassMethod,
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "empty calls array",
			callData:    packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{}),
			method:      &bypassMethod,
			expected:    nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCallFromMethod(tt.callData, tt.method)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil for callData len=%d, method=%v", len(tt.callData), tt.method != nil)
				return
			}

			require.NotNil(t, result, "Expected non-nil result")
			assert.Equal(t, tt.expected.Target, result.Target, "Target mismatch")
			assert.Equal(t, tt.expected.Value, result.Value, "Value mismatch")
			assert.Equal(t, tt.expected.Data, result.Data, "Data mismatch")
		})
	}
}

func TestExtractUnderlyingCall(t *testing.T) {
	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, "Failed to get RBACTimelock ABI")

	testCall := bindings.RBACTimelockCall{
		Target: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Value:  big.NewInt(1000),
		Data:   []byte{0x12, 0x34, 0x56, 0x78},
	}

	tests := []struct {
		name        string
		callData    []byte
		expected    *bindings.RBACTimelockCall
		shouldMatch bool
	}{
		{
			name:        "callData too short",
			callData:    []byte{0x12, 0x34}, // Less than 4 bytes
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "bypasserExecuteBatch with valid call",
			callData:    packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{testCall}),
			expected:    &testCall,
			shouldMatch: true,
		},
		{
			name:        "executeBatch with valid call",
			callData:    packExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{testCall}, common.Hash{}, common.Hash{}),
			expected:    &testCall,
			shouldMatch: true,
		},
		{
			name:        "unknown method selector",
			callData:    append([]byte{0xFF, 0xFF, 0xFF, 0xFF}, make([]byte, 100)...),
			expected:    nil,
			shouldMatch: false,
		},
		{
			name:        "empty callData",
			callData:    []byte{},
			expected:    nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUnderlyingCall(tt.callData)

			if !tt.shouldMatch {
				assert.Nil(t, result, "Expected nil for callData len=%d", len(tt.callData))
				return
			}

			require.NotNil(t, result, "Expected non-nil result")
			assert.Equal(t, tt.expected.Target, result.Target, "Target mismatch")
			assert.Equal(t, tt.expected.Value, result.Value, "Value mismatch")
			assert.Equal(t, tt.expected.Data, result.Data, "Data mismatch")
		})
	}
}

func TestGetUnderlyingRevertReason(t *testing.T) {
	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, "Failed to get RBACTimelock ABI")

	testCall := bindings.RBACTimelockCall{
		Target: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Value:  big.NewInt(1000),
		Data:   []byte{0x12, 0x34, 0x56, 0x78},
	}

	timelockAddr := common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0")
	callData := packBypasserExecuteBatch(t, timelockABI, []bindings.RBACTimelockCall{testCall})
	opts := &bind.TransactOpts{
		GasLimit: 1000000,
	}

	tests := []struct {
		name           string
		timelockAddr   common.Address
		callData       []byte
		opts           *bind.TransactOpts
		setupMock      func(*mocks.ContractDeployBackend)
		expectedResult string
		shouldDecode   bool
	}{
		{
			name:           "nil client",
			timelockAddr:   timelockAddr,
			callData:       callData,
			opts:           opts,
			setupMock:      nil, // nil client
			expectedResult: "",
			shouldDecode:   false,
		},
		{
			name:           "nil opts",
			timelockAddr:   timelockAddr,
			callData:       callData,
			opts:           nil,
			setupMock:      func(m *mocks.ContractDeployBackend) {},
			expectedResult: "",
			shouldDecode:   false,
		},
		{
			name:           "empty timelock address",
			timelockAddr:   common.Address{},
			callData:       callData,
			opts:           opts,
			setupMock:      func(m *mocks.ContractDeployBackend) {},
			expectedResult: "",
			shouldDecode:   false,
		},
		{
			name:           "empty callData",
			timelockAddr:   timelockAddr,
			callData:       []byte{},
			opts:           opts,
			setupMock:      func(m *mocks.ContractDeployBackend) {},
			expectedResult: "",
			shouldDecode:   false,
		},
		{
			name:         "call succeeds - no revert",
			timelockAddr: timelockAddr,
			callData:     callData,
			opts:         opts,
			setupMock: func(m *mocks.ContractDeployBackend) {
				// CallContract is called with specific parameters from extractUnderlyingCall
				m.On("CallContract", mock.Anything, mock.MatchedBy(func(msg any) bool {
					return true // Match any CallMsg
				}), mock.Anything).Return([]byte{}, nil).Maybe()
			},
			expectedResult: "",
			shouldDecode:   false,
		},
		{
			name:         "call reverts with decoded error",
			timelockAddr: timelockAddr,
			callData:     callData,
			opts:         opts,
			setupMock: func(m *mocks.ContractDeployBackend) {
				m.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("execution reverted: custom error 0x70de1b4b: aabbccdd")).Maybe()
			},
			expectedResult: "CallReverted(truncated)", // CallReverted with insufficient data returns truncated marker
			shouldDecode:   true,
		},
		{
			name:         "call reverts with plain string",
			timelockAddr: timelockAddr,
			callData:     callData,
			opts:         opts,
			setupMock: func(m *mocks.ContractDeployBackend) {
				// CallContract is called with specific parameters from extractUnderlyingCall
				// The call will have From=timelockAddr, To=testCall.Target, Value=testCall.Value, Data=testCall.Data
				m.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("execution reverted: revert: Ownable: caller is not the owner")).Maybe()
			},
			expectedResult: "Ownable: caller is not the owner",
			shouldDecode:   true,
		},
		{
			name:         "non-revert error",
			timelockAddr: timelockAddr,
			callData:     callData,
			opts:         opts,
			setupMock: func(m *mocks.ContractDeployBackend) {
				m.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("network error")).Maybe()
			},
			expectedResult: "",
			shouldDecode:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client ContractDeployBackend
			if tt.setupMock != nil {
				mockClient := mocks.NewContractDeployBackend(t)
				tt.setupMock(mockClient)
				client = mockClient
			}

			result := getUnderlyingRevertReason(context.Background(), tt.timelockAddr, tt.callData, tt.opts, client)

			// Verify mock expectations were met (if mock was set up)
			if tt.setupMock != nil {
				mockClient := client.(*mocks.ContractDeployBackend)
				mockClient.AssertExpectations(t)
			}

			if !tt.shouldDecode {
				assert.Empty(t, result, "Expected empty result for test: %s. Got: %q", tt.name, result)
			} else {
				assert.NotEmpty(t, result, "Expected non-empty result for test: %s", tt.name)
				if tt.expectedResult != "" {
					assert.Equal(t, tt.expectedResult, result, "Result mismatch for test: %s", tt.name)
				}
			}
		})
	}
}

// Helper functions for packing timelock method calls

// encodeErrorString ABI-encodes a string for Error(string) format.
// Uses go-ethereum's ABI encoder by creating a dummy function with string input.
func encodeErrorString(s string) []byte {
	// Create a dummy function ABI with a string parameter
	funcABI := `[{"name":"dummy","type":"function","inputs":[{"type":"string","name":"message"}]}]`
	parsedABI, err := abi.JSON(strings.NewReader(funcABI))
	if err != nil {
		// Fallback to manual encoding if ABI parsing fails
		return encodeErrorStringManual(s)
	}

	// Pack using the function's inputs (this gives us just the data, no selector needed)
	packed, err := parsedABI.Methods["dummy"].Inputs.Pack(s)
	if err != nil {
		// Fallback to manual encoding if packing fails
		return encodeErrorStringManual(s)
	}

	return packed
}

// encodeErrorStringManual manually encodes a string following ABI encoding rules.
// Format: offset (32 bytes) + length (32 bytes) + data (padded to 32 bytes)
func encodeErrorStringManual(s string) []byte {
	data := []byte(s)
	length := len(data)
	padding := (32 - (length % 32)) % 32

	encoded := make([]byte, 0, 32+32+length+padding)
	// Offset (32 bytes) - points to length field at offset 0x20
	encoded = append(encoded, make([]byte, 28)...)
	encoded = append(encoded, 0x20)

	// Length (32 bytes) - big-endian uint256
	encoded = append(encoded, make([]byte, 28)...)
	lengthBytes := make([]byte, 4)
	lengthBytes[3] = byte(length)
	lengthBytes[2] = byte(length >> 8)
	lengthBytes[1] = byte(length >> 16)
	lengthBytes[0] = byte(length >> 24)
	encoded = append(encoded, lengthBytes...)

	// String data with padding
	encoded = append(encoded, data...)
	encoded = append(encoded, make([]byte, padding)...)

	return encoded
}

func packBypasserExecuteBatch(t *testing.T, abi *abi.ABI, calls []bindings.RBACTimelockCall) []byte {
	t.Helper()
	method, exists := abi.Methods["bypasserExecuteBatch"]
	require.True(t, exists, "bypasserExecuteBatch method not found")

	data, err := method.Inputs.Pack(calls)
	require.NoError(t, err, "Failed to pack bypasserExecuteBatch")
	return append(method.ID, data...)
}

func packExecuteBatch(t *testing.T, abi *abi.ABI, calls []bindings.RBACTimelockCall, predecessor, salt common.Hash) []byte {
	t.Helper()
	method, exists := abi.Methods["executeBatch"]
	require.True(t, exists, "executeBatch method not found")

	data, err := method.Inputs.Pack(calls, predecessor, salt)
	require.NoError(t, err, "Failed to pack executeBatch")
	return append(method.ID, data...)
}
