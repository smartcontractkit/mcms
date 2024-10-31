package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
)

func TestEVMEncoder_HashOperation(t *testing.T) {
	t.Parallel()

	var (
		evmChainID    = uint64(1)
		chainSelector = mcms.ChainSelector(cselectors.EvmChainIdToChainSelector()[evmChainID])

		// Static argument values to HashOperation since they don't affect the test
		giveOpCount  = uint32(0)
		giveMetadata = mcms.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      "0x1",
		}
	)

	tests := []struct {
		name    string
		giveOp  mcms.ChainOperation
		want    string
		wantErr string
	}{
		{
			name: "success: hash operation",
			giveOp: mcms.ChainOperation{
				ChainSelector: chainSelector,
				Operation: NewEVMOperation(
					common.HexToAddress("0x2"),
					[]byte("data"),
					new(big.Int).SetUint64(1000000000000000000),
					"",
					[]string{},
				),
			},
			want: "0x4bd3162db447382fc6e5b2a8eb24e19e901feecb90c5071bac5deee3cb58cb97",
		},
		{
			name: "failure: cannot unmarshal additional fields",
			giveOp: mcms.ChainOperation{
				ChainSelector: chainSelector,
				Operation: mcms.Operation{
					AdditionalFields: []byte("invalid"),
				},
			},
			wantErr: "invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEVMEncoder(5, evmChainID, false)
			got, err := encoder.HashOperation(giveOpCount, giveMetadata, tt.giveOp)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got.Hex())
			}
		})
	}
}

func TestEVMEncoder_HashMetadata(t *testing.T) {
	t.Parallel()

	metadata := mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x1",
	}

	encoder := NewEVMEncoder(5, 1, false)
	got, err := encoder.HashMetadata(metadata)

	require.NoError(t, err)
	want := "0x3d030bfa3fcbbfa780ab87e0368f9487a271df7654cc2d1ba82f3be9d4933366"
	assert.Equal(t, want, got.Hex())
}

func TestEVMEncoder_ToGethOperation(t *testing.T) {
	t.Parallel()

	var (
		evmChainID    = uint64(1)
		chainSelector = mcms.ChainSelector(cselectors.EvmChainIdToChainSelector()[evmChainID])

		// Static argument values to ToGethOperation since they don't affect the test
		giveOpCount  = uint32(0)
		giveMetadata = mcms.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      "0x1",
		}
	)

	tests := []struct {
		name    string
		giveOp  mcms.ChainOperation
		want    bindings.ManyChainMultiSigOp
		wantErr string
	}{
		{
			name: "success: converts to a geth operations",
			giveOp: mcms.ChainOperation{
				ChainSelector: chainSelector,
				Operation: NewEVMOperation(
					common.HexToAddress("0x2"),
					[]byte("data"),
					new(big.Int).SetUint64(1000000000000000000),
					"",
					[]string{},
				),
			},
			want: bindings.ManyChainMultiSigOp{
				ChainId:  new(big.Int).SetUint64(1),
				MultiSig: common.HexToAddress("0x1"),
				Nonce:    new(big.Int).SetUint64(0),
				To:       common.HexToAddress("0x2"),
				Data:     []byte("data"),
				Value:    new(big.Int).SetUint64(1000000000000000000),
			},
		},
		{
			name: "failure: cannot unmarshal additional fields",
			giveOp: mcms.ChainOperation{
				ChainSelector: chainSelector,
				Operation: mcms.Operation{
					AdditionalFields: []byte("invalid"),
				},
			},
			wantErr: "invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEVMEncoder(5, evmChainID, false)
			got, err := encoder.ToGethOperation(giveOpCount, giveMetadata, tt.giveOp)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEVMEncoder_ToGethRootMetadata(t *testing.T) {
	t.Parallel()

	metadata := mcms.ChainMetadata{
		StartingOpCount: 0,
		MCMAddress:      "0x1",
	}

	encoder := NewEVMEncoder(5, 1, false)
	got := encoder.ToGethRootMetadata(metadata)

	want := bindings.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(1),
		MultiSig:             common.HexToAddress("0x1"),
		PreOpCount:           new(big.Int).SetUint64(0),
		PostOpCount:          new(big.Int).SetUint64(5),
		OverridePreviousRoot: false,
	}

	assert.Equal(t, want, got)
}
