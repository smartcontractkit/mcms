package evm

import (
	"context"
	"encoding/json"
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

const (
	errMsgOriginalError        = "original error"
	errMsgExecutionFailed      = "execution failed"
	errMsgRawRevertData        = "raw revert data"
	errMsgUnderlyingReason     = "underlying reason"
	errMsgRevertReason         = "revert reason"
	errMsgExecutionRevertedHex = "execution reverted: 0x1234567890abcdef"
	errMsgCustomError          = "execution reverted: custom error 0x70de1b4b: aabbccdd"
	errMsgExpectedNilFmt       = "Expected nil for input: %q"
	errMsgExpectedNonNilFmt    = "Expected non-nil result for input: %q"
	descriptionEmptyString     = "empty string"
	errMsgFailedTimelockABI    = "Failed to get RBACTimelock ABI"
)

type executionErrorErrorCase struct {
	name                string
	execErr             *ExecutionError
	expectedContains    []string
	expectedNotContains []string
}

var executionErrorErrorCases = []executionErrorErrorCase{
	{
		name: "with decoded underlying reason - highest priority",
		execErr: &ExecutionError{
			OriginalError:           errors.New(errMsgOriginalError),
			UnderlyingReasonRaw:     "0x1234",
			UnderlyingReasonDecoded: "decoded underlying revert reason",
			RevertReasonDecoded:     "decoded reason",
			RevertReasonRaw: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     []byte{0xaa, 0xbb, 0xcc},
			},
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
			errMsgUnderlyingReason + ": decoded underlying revert reason",
		},
		expectedNotContains: []string{
			errMsgRevertReason + ": decoded reason",
			errMsgRawRevertData,
		},
	},
	{
		name: "with raw underlying reason when decoded unavailable",
		execErr: &ExecutionError{
			OriginalError:       errors.New(errMsgOriginalError),
			UnderlyingReasonRaw: "underlying revert reason",
			RevertReasonDecoded: "decoded reason",
			RevertReasonRaw: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     []byte{0xaa, 0xbb, 0xcc},
			},
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
			errMsgUnderlyingReason + ": underlying revert reason",
		},
		expectedNotContains: []string{
			errMsgRevertReason + ": decoded reason",
			errMsgRawRevertData,
		},
	},
	{
		name: "with decoded revert reason - second priority",
		execErr: &ExecutionError{
			OriginalError:       errors.New(errMsgOriginalError),
			RevertReasonDecoded: "OutOfBoundsGroup",
			RevertReasonRaw: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     []byte{0xaa, 0xbb, 0xcc},
			},
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
			errMsgRevertReason + ": OutOfBoundsGroup",
		},
		expectedNotContains: []string{
			errMsgUnderlyingReason,
			errMsgRawRevertData,
		},
	},
	{
		name: "with raw revert reason - third priority",
		execErr: &ExecutionError{
			OriginalError: errors.New(errMsgOriginalError),
			RevertReasonRaw: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     []byte{0xaa, 0xbb, 0xcc},
			},
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
			errMsgRawRevertData,
			"12345678aabbcc",
		},
		expectedNotContains: []string{
			errMsgUnderlyingReason,
			errMsgRevertReason,
		},
	},
	{
		name: "with empty raw revert reason - selector only still shows raw data",
		execErr: &ExecutionError{
			OriginalError: errors.New(errMsgOriginalError),
			RevertReasonRaw: &CustomErrorData{
				Selector: [4]byte{0x12, 0x34, 0x56, 0x78},
				Data:     []byte{},
			},
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
			errMsgRawRevertData,
			"12345678",
		},
		expectedNotContains: []string{
			errMsgUnderlyingReason,
			errMsgRevertReason + ":",
		},
	},
	{
		name: "with nil raw revert reason - should not show raw data",
		execErr: &ExecutionError{
			OriginalError:   errors.New(errMsgOriginalError),
			RevertReasonRaw: nil,
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
		},
		expectedNotContains: []string{
			errMsgUnderlyingReason,
			errMsgRevertReason,
			errMsgRawRevertData,
		},
	},
	{
		name: "only original error - fallback case",
		execErr: &ExecutionError{
			OriginalError: errors.New(errMsgOriginalError),
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
		},
		expectedNotContains: []string{
			errMsgUnderlyingReason,
			errMsgRevertReason,
			errMsgRawRevertData,
		},
	},
	{
		name: "with CallReverted selector",
		execErr: &ExecutionError{
			OriginalError: errors.New(errMsgOriginalError),
			RevertReasonRaw: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     []byte{0xaa, 0xbb, 0xcc},
			},
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			errMsgOriginalError,
			errMsgRawRevertData,
			"70de1b4b",
		},
	},
	{
		name: "with decoded reason but no underlying reason",
		execErr: &ExecutionError{
			OriginalError:       errors.New("contract call failed"),
			RevertReasonDecoded: "InsufficientSigners",
		},
		expectedContains: []string{
			errMsgExecutionFailed,
			"contract call failed",
			errMsgRevertReason + ": InsufficientSigners",
		},
		expectedNotContains: []string{
			errMsgUnderlyingReason,
			errMsgRawRevertData,
		},
	},
}

func TestExecutionErrorError(t *testing.T) {
	t.Parallel()

	for _, tt := range executionErrorErrorCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

func TestExecutionErrorUnwrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New(errMsgOriginalError)
	execErr := &ExecutionError{
		OriginalError: originalErr,
	}

	unwrapped := execErr.Unwrap()
	require.Equal(t, originalErr, unwrapped, "Unwrap should return the original error")
}

func TestCustomErrorData(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			// Test HexSelector()
			hexSelector := tt.customErr.HexSelector()
			assert.Equal(t, tt.expectedHex, hexSelector, "HexSelector() mismatch")

			// Test Combined()
			combined := tt.customErr.Combined()
			assert.Equal(t, tt.expectedCombined, combined, "Combined() mismatch")
		})
	}
}

func TestCustomErrorDataJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal_and_unmarshal_with_data", func(t *testing.T) {
		t.Parallel()

		custom := &CustomErrorData{
			Selector: [4]byte{0xde, 0xad, 0xbe, 0xef},
			Data:     []byte{0xaa, 0xbb, 0xcc},
		}

		bytes, err := json.Marshal(custom)
		require.NoError(t, err)
		assert.JSONEq(t, `{"selector":"0xdeadbeef","data":"0xaabbcc"}`, string(bytes))

		var decoded CustomErrorData
		require.NoError(t, json.Unmarshal(bytes, &decoded))
		assert.Equal(t, custom.Selector, decoded.Selector)
		assert.Equal(t, custom.Data, decoded.Data)
	})

	t.Run("marshal_without_data", func(t *testing.T) {
		t.Parallel()

		custom := &CustomErrorData{
			Selector: [4]byte{0x70, 0xde, 0x1b, 0x4b},
		}

		bytes, err := json.Marshal(custom)
		require.NoError(t, err)
		assert.JSONEq(t, `{"selector":"0x70de1b4b"}`, string(bytes))
	})

	t.Run("unmarshal_invalid_hex_data", func(t *testing.T) {
		t.Parallel()

		var decoded CustomErrorData
		err := json.Unmarshal([]byte(`{"selector":"0xdeadbeef","data":"0xzz"}`), &decoded)
		require.Error(t, err)
	})
}

func TestExecutionErrorJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal_strips_original_error", func(t *testing.T) {
		t.Parallel()

		execErr := &ExecutionError{
			RevertReasonRaw: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     []byte{0x01, 0x02},
			},
			RevertReasonDecoded:     "CallReverted(truncated)",
			UnderlyingReasonRaw:     "0xdeadbeef",
			UnderlyingReasonDecoded: "OutOfBoundsGroup()",
			OriginalError:           errors.New("execution reverted: custom error"),
		}

		bytes, err := json.Marshal(execErr)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(bytes, &parsed))

		require.Equal(t, "execution reverted: custom error", parsed["OriginalError"])
		require.Equal(t, "CallReverted(truncated)", parsed["RevertReasonDecoded"])
		require.Equal(t, "OutOfBoundsGroup()", parsed["UnderlyingReasonDecoded"])
	})

	t.Run("unmarshal_restores_original_error", func(t *testing.T) {
		t.Parallel()

		payload := `{
			"Transaction": null,
			"RevertReasonRaw": {"selector": "0x70de1b4b", "data": "0x"},
			"RevertReasonDecoded": "CallReverted(truncated)",
			"UnderlyingReasonRaw": "0xdeadbeef",
			"UnderlyingReasonDecoded": "OutOfBoundsGroup()",
			"OriginalError": "execution reverted: custom error"
		}`

		var execErr ExecutionError
		require.NoError(t, json.Unmarshal([]byte(payload), &execErr))

		require.EqualError(t, execErr.OriginalError, "execution reverted: custom error")
		require.Equal(t, "CallReverted(truncated)", execErr.RevertReasonDecoded)
		require.Equal(t, "OutOfBoundsGroup()", execErr.UnderlyingReasonDecoded)
		require.NotNil(t, execErr.RevertReasonRaw)
		assert.Equal(t, CallRevertedSelector, execErr.RevertReasonRaw.Selector)
	})
}

func TestExtractHexEncodedRevertData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		errStr      string
		expected    []byte
		shouldMatch bool
	}{
		{
			name:        "valid hex string",
			errStr:      errMsgExecutionRevertedHex,
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
			name:        descriptionEmptyString,
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
			t.Parallel()

			result := extractHexEncodedRevertData(tt.errStr)

			if !tt.shouldMatch {
				assert.Nil(t, result, errMsgExpectedNilFmt, tt.errStr)
				return
			}

			require.NotNil(t, result, errMsgExpectedNonNilFmt, tt.errStr)
			assert.Equal(t, tt.expected, result, "Extracted hex data mismatch for input: %q", tt.errStr)
		})
	}
}

func TestExtractBytesArrayRevertData(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			result := extractBytesArrayRevertData(tt.errStr)

			if !tt.shouldMatch {
				assert.Nil(t, result, errMsgExpectedNilFmt, tt.errStr)
				return
			}

			require.NotNil(t, result, errMsgExpectedNonNilFmt, tt.errStr)
			assert.Equal(t, tt.expected, result, "Extracted bytes mismatch for input: %q", tt.errStr)
		})
	}
}

func TestExtractCustomErrorRevertData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		errStr      string
		expected    *CustomErrorData
		shouldMatch bool
	}{
		{
			name:   "valid custom error",
			errStr: errMsgCustomError,
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
			name:        descriptionEmptyString,
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
			t.Parallel()

			result := extractCustomErrorRevertData(tt.errStr)

			if !tt.shouldMatch {
				assert.Nil(t, result, errMsgExpectedNilFmt, tt.errStr)
				return
			}

			require.NotNil(t, result, errMsgExpectedNonNilFmt, tt.errStr)
			assert.Equal(t, tt.expected.Selector, result.Selector, "Selector mismatch for input: %q", tt.errStr)
			assert.Equal(t, tt.expected.Data, result.Data, "Data mismatch for input: %q", tt.errStr)
		})
	}
}

func TestParseBytesFromString(t *testing.T) {
	t.Parallel()

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
			name:     descriptionEmptyString,
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
			t.Parallel()

			result := parseBytesFromString(tt.input)
			assert.Equal(t, tt.expected, result, "Parsed bytes mismatch for input: %q", tt.input)
		})
	}
}

func TestExtractRevertReasonFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		err              error
		expectedRawData  []byte
		hasCustomError   bool
		expectedSelector [4]byte
		shouldHaveData   bool
	}{
		{
			name:            "nil error",
			err:             nil,
			expectedRawData: nil,
			hasCustomError:  false,
			shouldHaveData:  false,
		},
		{
			name:             "custom error format - highest priority",
			err:              errors.New(errMsgCustomError),
			expectedRawData:  append(CallRevertedSelector[:], common.FromHex("0xaabbccdd")...),
			hasCustomError:   true,
			expectedSelector: CallRevertedSelector,
			shouldHaveData:   true,
		},
		{
			name:            "CallReverted format with bytes array",
			err:             errors.New("contract error: error -`CallReverted` args [[8 195 121 160]]"),
			expectedRawData: []byte{8, 195, 121, 160},
			hasCustomError:  false,
			shouldHaveData:  true,
		},
		{
			name:            "hex-encoded format",
			err:             errors.New(errMsgExecutionRevertedHex),
			expectedRawData: common.FromHex("0x1234567890abcdef"),
			hasCustomError:  false,
			shouldHaveData:  true,
		},
		{
			name:            "bytes array format (without CallReverted)",
			err:             errors.New("error with bytes [[8 195 121 160]]"),
			expectedRawData: []byte{8, 195, 121, 160},
			hasCustomError:  false,
			shouldHaveData:  true,
		},
		{
			name:            "plain error without revert data",
			err:             errors.New("some generic error"),
			expectedRawData: nil,
			hasCustomError:  false,
			shouldHaveData:  false,
		},
		{
			name:            "invalid formats fall through",
			err:             errors.New("CallReverted but no bytes array"),
			expectedRawData: nil,
			hasCustomError:  false,
			shouldHaveData:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

	// Error(string) selector: keccak256("Error(string)")[:4] = 0x08c379a0
	errorStringSelector := [4]byte{0x08, 0xc3, 0x79, 0xa0}

	tests := []struct {
		name           string
		customErr      *CustomErrorData
		expectedResult string
		shouldDecode   bool
		description    string
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
			expectedResult: "",
			shouldDecode:   false,
			description:    "CallReverted with empty data",
		},
		{
			name: "CallReverted with valid data",
			customErr: &CustomErrorData{
				Selector: CallRevertedSelector,
				Data:     []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			expectedResult: "CallReverted(truncated)",
			shouldDecode:   true,
			description:    "CallReverted with insufficient data returns truncated marker",
		},
		{
			name: "Error(string) with valid string data",
			customErr: &CustomErrorData{
				Selector: errorStringSelector,
				Data:     encodeErrorString("test error message"),
			},
			expectedResult: "test error message",
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
			shouldDecode:   false,
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
				Data:     []byte{0x00, 0x00, 0x00, 0x00},
			},
			expectedResult: "",
			shouldDecode:   false,
			description:    "Error(string) with malformed/incomplete data should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

			result := formatDecodedError(tt.errorName, tt.decodedValues)
			assert.Equal(t, tt.expectedResult, result, "Format mismatch for errorName=%q, values=%v", tt.errorName, tt.decodedValues)
		})
	}
}

func TestExtractRevertDataFromCallError(t *testing.T) {
	t.Parallel()

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
			err:         errors.New(errMsgExecutionRevertedHex),
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
			t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

	// Get the real timelock ABI for testing
	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, errMsgFailedTimelockABI)

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
			callData:    []byte{0x12, 0x34},
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
			callData:    append(bypassMethod.ID, make([]byte, 100)...),
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
			t.Parallel()

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
	t.Parallel()

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, errMsgFailedTimelockABI)

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
			t.Parallel()

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
	t.Parallel()

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err, errMsgFailedTimelockABI)

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
		name            string
		timelockAddr    common.Address
		callData        []byte
		opts            *bind.TransactOpts
		setupMock       func(*mocks.ContractDeployBackend)
		expectedRaw     string
		expectedDecoded string
	}{
		{
			name:            "nil client",
			timelockAddr:    timelockAddr,
			callData:        callData,
			opts:            opts,
			setupMock:       nil, // nil client
			expectedRaw:     "",
			expectedDecoded: "",
		},
		{
			name:            "nil opts",
			timelockAddr:    timelockAddr,
			callData:        callData,
			opts:            nil,
			setupMock:       func(m *mocks.ContractDeployBackend) {},
			expectedRaw:     "",
			expectedDecoded: "",
		},
		{
			name:            "empty timelock address",
			timelockAddr:    common.Address{},
			callData:        callData,
			opts:            opts,
			setupMock:       func(m *mocks.ContractDeployBackend) {},
			expectedRaw:     "",
			expectedDecoded: "",
		},
		{
			name:            "empty callData",
			timelockAddr:    timelockAddr,
			callData:        []byte{},
			opts:            opts,
			setupMock:       func(m *mocks.ContractDeployBackend) {},
			expectedRaw:     "",
			expectedDecoded: "",
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
			expectedRaw:     "",
			expectedDecoded: "",
		},
		{
			name:         "call reverts with decoded error",
			timelockAddr: timelockAddr,
			callData:     callData,
			opts:         opts,
			setupMock: func(m *mocks.ContractDeployBackend) {
				m.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New(errMsgCustomError)).Maybe()
			},
			expectedRaw:     "0x70de1b4b",
			expectedDecoded: "CallReverted(truncated)", // CallReverted with insufficient data returns truncated marker
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
			expectedRaw:     "Ownable: caller is not the owner",
			expectedDecoded: "Ownable: caller is not the owner",
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
			expectedRaw:     "",
			expectedDecoded: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var client ContractDeployBackend
			if tt.setupMock != nil {
				mockClient := mocks.NewContractDeployBackend(t)
				tt.setupMock(mockClient)
				client = mockClient
			}

			rawReason, decodedReason := getUnderlyingRevertReason(context.Background(), tt.timelockAddr, tt.callData, tt.opts, client)

			// Verify mock expectations were met (if mock was set up)
			if tt.setupMock != nil {
				mockClient := client.(*mocks.ContractDeployBackend)
				mockClient.AssertExpectations(t)
			}

			assert.Equal(t, tt.expectedRaw, rawReason, "Raw reason mismatch for test: %s", tt.name)
			assert.Equal(t, tt.expectedDecoded, decodedReason, "Decoded reason mismatch for test: %s", tt.name)
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

func packBypasserExecuteBatch(t *testing.T, timelockAbi *abi.ABI, calls []bindings.RBACTimelockCall) []byte {
	t.Helper()
	method, exists := timelockAbi.Methods["bypasserExecuteBatch"]
	require.True(t, exists, "bypasserExecuteBatch method not found")

	data, err := method.Inputs.Pack(calls)
	require.NoError(t, err, "Failed to pack bypasserExecuteBatch")

	return append(method.ID, data...)
}

func packExecuteBatch(t *testing.T, timelockAbi *abi.ABI, calls []bindings.RBACTimelockCall, predecessor, salt common.Hash) []byte {
	t.Helper()
	method, exists := timelockAbi.Methods["executeBatch"]
	require.True(t, exists, "executeBatch method not found")

	data, err := method.Inputs.Pack(calls, predecessor, salt)
	require.NoError(t, err, "Failed to pack executeBatch")

	return append(method.ID, data...)
}
