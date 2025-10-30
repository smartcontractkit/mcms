package ton

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
)

func TestEncoder_HashOperation(t *testing.T) {
	t.Parallel()

	var (
		// Static argument values to HashOperation since they don't affect the test
		giveOpCount  = uint32(0)
		giveMetadata = types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
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
				ChainSelector: chaintest.Chain7Selector,
				Transaction: must(NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
					new(big.Int).SetUint64(1000000000000000000),
					"",
					[]string{},
				)),
			},
			want: "0xc6d17bac676615dd0e6e854e41066557366c71bed4c75401741e231050196361",
		},
		{
			name: "failure: cannot unmarshal additional fields",
			giveOp: types.Operation{
				ChainSelector: chaintest.Chain7Selector,
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid"),
				},
			},
			wantErr: "failed to convert operation: invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEncoder(chaintest.Chain7Selector, 5, false)
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
			giveSelector: chaintest.Chain7Selector,
			giveMeta: types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
			want: "0x5e0ca25000083f3d564a2a5db871488550c1915b42fda858df95018a40e881ff",
		},
		{
			name:         "failure: could not get TON chain id",
			giveSelector: chaintest.ChainInvalidSelector,
			giveMeta:     types.ChainMetadata{},
			wantErr:      "failed to convert to root metadata: invalid chain ID: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoder := NewEncoder(tt.giveSelector, 1, false)
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

func TestEncoder_ToOperation(t *testing.T) {
	t.Parallel()

	var (
		chainID       = int32(-217)
		chainSelector = types.ChainSelector(cselectors.TonChainIdToChainSelector()[chainID])

		// Static argument values to ToGethOperation since they don't affect the test
		giveOpCount  = uint32(0)
		giveMetadata = types.ChainMetadata{
			StartingOpCount: 0,
			MCMAddress:      "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
		}
	)

	tests := []struct {
		name         string
		giveSelector types.ChainSelector
		giveOp       types.Operation
		want         mcms.Op
		wantErr      string
	}{
		{
			name:         "success: converts to a geth operations",
			giveSelector: chaintest.Chain7Selector,
			giveOp: types.Operation{
				ChainSelector: chainSelector,
				Transaction: must(NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
					new(big.Int).SetUint64(1000000000000000000),
					"",
					[]string{},
				)),
			},
			want: mcms.Op{
				ChainID:  new(big.Int).SetInt64(int64(chaintest.Chain7ToniID)),
				MultiSig: address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
				Nonce:    uint64(0),
				To:       address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
				// Notice: we wrap in BOC as it decodes differently to pass the equality test
				// -  refs: ([]*cell.Cell) <nil>
				// +  refs: ([]*cell.Cell) {}
				Data:  must(cell.FromBOC(cell.BeginCell().MustStoreBinarySnake([]byte("data")).EndCell().ToBOC())),
				Value: tlb.MustFromTON("1000000000"),
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
			giveSelector: chaintest.Chain7Selector,
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

			encoder := NewEncoder(tt.giveSelector, 5, false)
			got, err := encoder.(OperationEncoder[mcms.Op]).ToOperation(giveOpCount, giveMetadata, tt.giveOp)

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

func TestEncoder_ToRootMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector types.ChainSelector
		giveMetadata types.ChainMetadata
		want         mcms.RootMetadata
		wantErr      string
	}{
		{
			name:         "success: converts to a geth root metadata",
			giveSelector: chaintest.Chain7Selector,
			giveMetadata: types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
			want: mcms.RootMetadata{
				ChainID:              new(big.Int).SetInt64(int64(chaintest.Chain7ToniID)),
				MultiSig:             address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
				PreOpCount:           uint64(0),
				PostOpCount:          uint64(5),
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

	txCount := uint64(5)
	for _, tt := range tests {
		encoder := NewEncoder(tt.giveSelector, txCount, false)
		got, err := encoder.(RootMetadataEncoder[mcms.RootMetadata]).ToRootMetadata(tt.giveMetadata)

		if tt.wantErr != "" {
			require.EqualError(t, err, tt.wantErr)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		}
	}
}
