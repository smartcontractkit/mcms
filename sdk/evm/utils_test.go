package evm

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

func Test_TransformHashes(t *testing.T) {
	t.Parallel()

	hashes := []common.Hash{
		common.HexToHash("0x1"),
		common.HexToHash("0x2"),
	}

	want := [][32]byte{
		{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x01,
		},
		{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x02,
		},
	}

	got := transformHashes(hashes)
	assert.Equal(t, want, got)
}

func Test_TransformSignatures(t *testing.T) {
	t.Parallel()

	got := transformSignatures([]types.Signature{
		{
			R: common.HexToHash("0x1234567890abcdef"),
			S: common.HexToHash("0xfedcba0987654321"),
			V: 27,
		},
	})

	assert.Equal(t, []bindings.ManyChainMultiSigSignature{
		{
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
		},
	}, got)
}

func Test_toGethSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give types.Signature
		want bindings.ManyChainMultiSigSignature
	}{
		{
			name: "success",
			give: types.Signature{
				R: common.HexToHash("0x1234567890abcdef"),
				S: common.HexToHash("0xfedcba0987654321"),
				V: 27,
			},
			want: bindings.ManyChainMultiSigSignature{
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
			},
		},
		{
			name: "success: V < EthereumSignatureVThreshold",
			give: types.Signature{
				R: common.HexToHash("0x1234567890abcdef"),
				S: common.HexToHash("0xfedcba0987654321"),
				V: 0,
			},
			want: bindings.ManyChainMultiSigSignature{
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := toGethSignature(tt.give)
			assert.Equal(t, tt.want, result)
		})
	}
}

func Test_ABIEncode(t *testing.T) {
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

			got, err := ABIEncode(tt.giveABI, tt.giveValues...)

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

func Test_ABIDecode(t *testing.T) {
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

			got, err := abiDecode(tt.giveABI, data)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
