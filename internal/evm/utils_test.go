package evm

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToGethSignature(t *testing.T) {
	t.Parallel()

	sig := mcms.Signature{
		R: common.HexToHash("0x1234567890abcdef"),
		S: common.HexToHash("0xfedcba0987654321"),
		V: 27,
	}

	expected := bindings.ManyChainMultiSigSignature{
		R: [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
		S: [32]byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0xfe, 0xdc, 0xba, 0x09, 0x87, 0x65, 0x43, 0x21},
		V: 27,
	}

	result := ToGethSignature(sig)
	assert.Equal(t, expected, result)
}

func TestABIDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		abiStr      string
		dataHex     string
		expected    []any
		expectError bool
	}{
		{
			name:    "Decode single uint256",
			abiStr:  `[{"type":"uint256"}]`,
			dataHex: "000000000000000000000000000000000000000000000000000000000000001e", // 30 in uint256
			expected: []any{
				big.NewInt(30),
			},
			expectError: false,
		},
		{
			name:        "Decode address",
			abiStr:      `[{"type":"address"}]`,
			dataHex:     "0000000000000000000000005b38da6a701c568545dcfcb03fcb875f56beddc4", // valid Ethereum address
			expected:    []any{common.HexToAddress("0x5b38da6a701c568545dcfcb03fcb875f56beddc4")},
			expectError: false,
		},
		{
			name:   "Decode string",
			abiStr: `[{"type":"string"}]`,
			dataHex: "0000000000000000000000000000000000000000000000000000000000000020" + // offset (32 bytes)
				"000000000000000000000000000000000000000000000000000000000000000b" + // string length (11 bytes)
				"48656c6c6f20576f726c64000000000000000000000000000000000000000000", // "Hello World"
			expected:    []any{"Hello World"},
			expectError: false,
		},
		{
			name:        "Invalid data",
			abiStr:      `[{"type":"uint256"}]`,
			dataHex:     "00000000000000000000000000000000", // Too short for uint256
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid ABI string",
			abiStr:      `[{"type":"invalid"}]`,                                             // Invalid ABI type
			dataHex:     "000000000000000000000000000000000000000000000000000000000000001e", // 30 in uint256
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := hex.DecodeString(tt.dataHex)
			require.NoError(t, err)

			result, err := ABIDecode(tt.abiStr, data)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
