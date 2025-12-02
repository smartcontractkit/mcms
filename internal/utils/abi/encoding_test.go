package abi

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Encode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		giveABI    string
		giveValues []any
		want       string
		wantError  bool
	}{
		{
			name:    "success: encode single uint256",
			giveABI: `[{"type":"uint256"}]`,
			giveValues: []any{
				big.NewInt(30), // 30 in uint256
			},
			want: "000000000000000000000000000000000000000000000000000000000000001e",
		},
		{
			name:       "success: encode address",
			giveABI:    `[{"type":"address"}]`,
			giveValues: []any{common.HexToAddress("0x5b38da6a701c568545dcfcb03fcb875f56beddc4")}, // valid Ethereum address,
			want:       "0000000000000000000000005b38da6a701c568545dcfcb03fcb875f56beddc4",
		},
		{
			name:       "success: encode string",
			giveABI:    `[{"type":"string"}]`,
			giveValues: []any{"Hello World"},
			want: "0000000000000000000000000000000000000000000000000000000000000020" + // offset (32 bytes)
				"000000000000000000000000000000000000000000000000000000000000000b" + // string length (11 bytes)
				"48656c6c6f20576f726c64000000000000000000000000000000000000000000", // "Hello World"
		},
		{
			name:      "failure: invalid ABI string",
			giveABI:   `[{"type":"invalid"}]`, // Invalid ABI type
			wantError: true,
		},
		{
			name:       "failure: invalid values",
			giveABI:    `[{"type":"uint256"}]`,
			giveValues: []any{}, // No values
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Encode(tt.giveABI, tt.giveValues...)

			if tt.wantError {
				require.Error(t, err)
			} else {
				wantBytes, err := hex.DecodeString(tt.want)
				require.NoError(t, err)

				require.NoError(t, err)
				assert.Equal(t, wantBytes, got)
			}
		})
	}
}

func Test_Decode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		giveABI   string
		giveData  string
		want      []any
		wantError bool
	}{
		{
			name:     "success: decode single uint256",
			giveABI:  `[{"type":"uint256"}]`,
			giveData: "000000000000000000000000000000000000000000000000000000000000001e", // 30 in uint256
			want: []any{
				big.NewInt(30),
			},
		},
		{
			name:     "success: decode address",
			giveABI:  `[{"type":"address"}]`,
			giveData: "0000000000000000000000005b38da6a701c568545dcfcb03fcb875f56beddc4", // valid Ethereum address
			want:     []any{common.HexToAddress("0x5b38da6a701c568545dcfcb03fcb875f56beddc4")},
		},
		{
			name:    "success: decode string",
			giveABI: `[{"type":"string"}]`,
			giveData: "0000000000000000000000000000000000000000000000000000000000000020" + // offset (32 bytes)
				"000000000000000000000000000000000000000000000000000000000000000b" + // string length (11 bytes)
				"48656c6c6f20576f726c64000000000000000000000000000000000000000000", // "Hello World"
			want:      []any{"Hello World"},
			wantError: false,
		},
		{
			name:      "failure: invalid data",
			giveABI:   `[{"type":"uint256"}]`,
			giveData:  "00000000000000000000000000000000", // Too short for uint256
			wantError: true,
		},
		{
			name:      "failure: invalid ABI string",
			giveABI:   `[{"type":"invalid"}]`,                                             // Invalid ABI type
			giveData:  "000000000000000000000000000000000000000000000000000000000000001e", // 30 in uint256
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := hex.DecodeString(tt.giveData)
			require.NoError(t, err)

			got, err := Decode(tt.giveABI, data)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
