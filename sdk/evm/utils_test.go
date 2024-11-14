package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

func Test_transformHashes(t *testing.T) {
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

func Test_transformSignatures(t *testing.T) {
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

func Test_getEVMChainID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		giveSel   types.ChainSelector
		giveIsSim bool
		want      uint64
		wantErr   bool
	}{
		{
			name:      "success: valid chain selector",
			giveSel:   chaintest.Chain2Selector,
			giveIsSim: false,
			want:      11155111,
		},
		{
			name:      "success: simulated chain",
			giveSel:   0,
			giveIsSim: true,
			want:      SimulatedEVMChainID,
		},
		{
			name:      "error: invalid chain selector",
			giveSel:   types.ChainSelector(9999),
			giveIsSim: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getEVMChainID(tt.giveSel, tt.giveIsSim)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
