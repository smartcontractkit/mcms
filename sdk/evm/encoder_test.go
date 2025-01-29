package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

func TestEncoder_HashOperation(t *testing.T) {
	t.Parallel()

	var (
		// Static argument values to HashOperation since they don't affect the test
		giveOpCount  = uint32(0)
		giveMetadata = types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      "0x1",
		}
	)

	tests := []struct {
		name    string
		giveOp  types.Operation
		want    string
		wantErr string
	}{
		{
			name: "success: hash operation",
			giveOp: types.Operation{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: NewTransaction(
					common.HexToAddress("0x2"),
					[]byte("data"),
					new(big.Int).SetUint64(1000000000000000000),
					"",
					[]string{},
				),
			},
			want: "0x689c68d31325bd84109a05d2319a683560f9773962d58d813915102210f571ef",
		},
		{
			name: "failure: cannot unmarshal additional fields",
			giveOp: types.Operation{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid"),
				},
			},
			wantErr: "invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEncoder(chaintest.Chain1Selector, 5, false, false)
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

func TestEncoder_HashMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector types.ChainSelector
		giveMeta     types.ChainMetadata
		want         string
		wantErr      string
	}{
		{
			name:         "success: hash metadata",
			giveSelector: chaintest.Chain1Selector,
			giveMeta: types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      "0x1",
			},
			want: "0x2402fdba934e2c405ce8e34c096eaa95a9d34291ca945c7ccede4bec8160cac0",
		},
		{
			name:         "failure: could not get evm chain id",
			giveSelector: chaintest.ChainInvalidSelector,
			giveMeta:     types.ChainMetadata{},
			wantErr:      "invalid chain ID: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEncoder(tt.giveSelector, 1, false, false)
			got, err := encoder.HashMetadata(tt.giveMeta)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got.Hex())
			}
		})
	}
}

func TestEncoder_ToGethOperation(t *testing.T) {
	t.Parallel()

	var (
		evmChainID    = uint64(1)
		chainSelector = types.ChainSelector(cselectors.EvmChainIdToChainSelector()[evmChainID])

		// Static argument values to ToGethOperation since they don't affect the test
		giveOpCount  = uint32(0)
		giveMetadata = types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      "0x1",
		}
	)

	tests := []struct {
		name         string
		giveSelector types.ChainSelector
		giveOp       types.Operation
		want         bindings.ManyChainMultiSigOp
		wantErr      string
	}{
		{
			name:         "success: converts to a geth operations",
			giveSelector: chaintest.Chain1Selector,
			giveOp: types.Operation{
				ChainSelector: chainSelector,
				Transaction: NewTransaction(
					common.HexToAddress("0x2"),
					[]byte("data"),
					new(big.Int).SetUint64(1000000000000000000),
					"",
					[]string{},
				),
			},
			want: bindings.ManyChainMultiSigOp{
				ChainId:  new(big.Int).SetUint64(chaintest.Chain1EVMID),
				MultiSig: common.HexToAddress("0x1"),
				Nonce:    new(big.Int).SetUint64(0),
				To:       common.HexToAddress("0x2"),
				Data:     []byte("data"),
				Value:    new(big.Int).SetUint64(1000000000000000000),
			},
		},
		{
			name:         "failure: invalid chain selector",
			giveSelector: chaintest.ChainInvalidSelector,
			giveOp:       types.Operation{},
			wantErr:      "invalid chain ID: 0",
		},
		{
			name:         "failure: cannot unmarshal additional fields",
			giveSelector: chaintest.Chain1Selector,
			giveOp: types.Operation{
				ChainSelector: chainSelector,
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid"),
				},
			},
			wantErr: "invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEncoder(tt.giveSelector, 5, false, false)
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

func TestEncoder_ToGethRootMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector types.ChainSelector
		giveMetadata types.ChainMetadata
		want         bindings.ManyChainMultiSigRootMetadata
		wantErr      string
	}{
		{
			name:         "success: converts to a geth root metadata",
			giveSelector: chaintest.Chain1Selector,
			giveMetadata: types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      "0x1",
			},
			want: bindings.ManyChainMultiSigRootMetadata{
				ChainId:              new(big.Int).SetUint64(chaintest.Chain1EVMID),
				MultiSig:             common.HexToAddress("0x1"),
				PreOpCount:           new(big.Int).SetUint64(0),
				PostOpCount:          new(big.Int).SetUint64(5),
				OverridePreviousRoot: false,
			},
		},
		{
			name:         "faiure: invalid chain selector",
			giveSelector: chaintest.ChainInvalidSelector,
			giveMetadata: types.ChainMetadata{},
			wantErr:      "invalid chain ID: 0",
		},
	}

	for _, tt := range tests {
		encoder := NewEncoder(tt.giveSelector, 5, false, false)

		got, err := encoder.ToGethRootMetadata(tt.giveMetadata)

		if tt.wantErr != "" {
			require.EqualError(t, err, tt.wantErr)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		}
	}
}
