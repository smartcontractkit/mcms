package ton_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/common"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/mcms/sdk/ton"
)

func TestTimelockConverter_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	timelockAddress := "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"
	mcmAddress := "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"
	zeroHash := common.Hash{}

	testOp := types.BatchOperation{
		Transactions: []types.Transaction{must(ton.NewTransaction(
			address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
			cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
			new(big.Int).SetUint64(1000),
			bindings.ShortTimelock,
			semver.MustParse("0.0.0"),
			bindings.TypeTimelock,
			[]string{"tag1", "tag2"},
		))},
		ChainSelector: types.ChainSelector(chainsel.TON_TESTNET.Selector),
	}

	testCases := []struct {
		name           string
		metadata       types.ChainMetadata
		op             types.BatchOperation
		delay          string
		operation      types.TimelockAction
		predecessor    common.Hash
		salt           common.Hash
		wantErr        string
		expectedOpType string
	}{
		{
			name:           "Schedule operation",
			op:             testOp,
			delay:          "1h",
			operation:      types.TimelockActionSchedule,
			predecessor:    zeroHash,
			salt:           zeroHash,
			expectedOpType: bindings.ShortTimelock,
		},
		{
			name:           "Cancel operation",
			op:             testOp,
			delay:          "1h",
			operation:      types.TimelockActionCancel,
			predecessor:    zeroHash,
			salt:           zeroHash,
			expectedOpType: bindings.ShortTimelock,
		},
		{
			name:           "Bypass operation",
			op:             testOp,
			delay:          "1h",
			operation:      types.TimelockActionBypass,
			predecessor:    zeroHash,
			salt:           zeroHash,
			expectedOpType: bindings.ShortTimelock,
		},
		{
			name:           "Invalid operation",
			op:             testOp,
			delay:          "1h",
			operation:      types.TimelockAction("invalid"),
			predecessor:    zeroHash,
			salt:           zeroHash,
			wantErr:        sdkerrors.NewInvalidTimelockOperationError("invalid").Error(),
			expectedOpType: "",
		},
		{
			name: "Invalid additional fields",
			op: types.BatchOperation{
				Transactions: []types.Transaction{{
					OperationMetadata: types.OperationMetadata{ContractType: bindings.ShortTimelock},
					To:                timelockAddress,
					Data:              []byte("0x1234"),
					AdditionalFields:  []byte("invalid"),
				}},
				ChainSelector: types.ChainSelector(chainsel.TON_TESTNET.Selector),
			},
			delay:       "1h",
			operation:   types.TimelockActionSchedule,
			predecessor: zeroHash,
			salt:        zeroHash,
			wantErr:     "failed to unmarshal TON additional fields: invalid character 'i' looking for beginning of value",
		},
		{
			name: "Invalid address in transaction",
			op: types.BatchOperation{
				Transactions: []types.Transaction{{
					OperationMetadata: types.OperationMetadata{ContractType: bindings.ShortTimelock},
					To:                "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-", // invalid address
					Data:              []byte("0x1234"),
					AdditionalFields:  []byte("{\"value\":1000}"),
				}},
				ChainSelector: types.ChainSelector(chainsel.TON_TESTNET.Selector),
			},
			delay:       "1h",
			operation:   types.TimelockActionSchedule,
			predecessor: zeroHash,
			salt:        zeroHash,
			wantErr:     "failed to convert batch to calls: invalid target address: incorrect address data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			converter := ton.NewTimelockConverter(ton.DefaultSendAmount)
			chainOperations, operationID, err := converter.ConvertBatchToChainOperations(
				ctx,
				tc.metadata,
				tc.op,
				timelockAddress,
				mcmAddress,
				types.MustParseDuration(tc.delay),
				tc.operation,
				tc.predecessor,
				tc.salt,
			)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, common.Hash{}, operationID)
				assert.Len(t, chainOperations, 1)
				assert.Equal(t, timelockAddress, chainOperations[0].Transaction.To)
				assert.Equal(t, tc.op.ChainSelector, chainOperations[0].ChainSelector)
				assert.Equal(t, tc.expectedOpType, chainOperations[0].Transaction.ContractType)
			}
		})
	}
}

func TestOperationID(t *testing.T) {
	t.Parallel()

	defaultTransaction := func() types.Transaction {
		return must(ton.NewTransaction(
			address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
			cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
			new(big.Int).SetUint64(1000),
			bindings.ShortTimelock,
			semver.MustParse("0.0.0"),
			bindings.TypeTimelock,
			[]string{},
		))
	}

	tests := []struct {
		name        string
		batchOp     types.BatchOperation
		action      types.TimelockAction
		predecessor common.Hash
		salt        common.Hash
		want        common.Hash
		wantErr     string
	}{
		{
			name: "success",
			batchOp: types.BatchOperation{
				ChainSelector: types.ChainSelector(chainsel.TON_TESTNET.Selector),
				Transactions:  []types.Transaction{defaultTransaction()},
			},
			action:      types.TimelockActionSchedule,
			predecessor: common.HexToHash("0x0123"),
			salt:        common.HexToHash("0xabcd"),
			want:        common.HexToHash("0xe158da9116bc9c70be25f6b3cfb7a2c0023f82fc4e60985d64148024a19d1609"),
		},
		{
			name: "failure: bad additional fields",
			batchOp: types.BatchOperation{
				ChainSelector: types.ChainSelector(chainsel.TON_TESTNET.Selector),
				Transactions: []types.Transaction{
					func() types.Transaction {
						tx := defaultTransaction()
						tx.AdditionalFields = []byte("invalid")
						return tx //nolint:nlreturn
					}(),
				},
			},
			action:      types.TimelockActionSchedule,
			predecessor: common.HexToHash("0x0123"),
			salt:        common.HexToHash("0xabcd"),
			wantErr:     "failed to unmarshal TON additional fields: invalid character",
		},
		{
			name: "failure: bad To address",
			batchOp: types.BatchOperation{
				ChainSelector: types.ChainSelector(chainsel.TON_TESTNET.Selector),
				Transactions: []types.Transaction{
					func() types.Transaction {
						tx := defaultTransaction()
						tx.To = "invalid address"
						return tx //nolint:nlreturn
					}(),
				},
			},
			action:      types.TimelockActionSchedule,
			predecessor: common.HexToHash("0x0123"),
			salt:        common.HexToHash("0xabcd"),
			wantErr:     "failed to convert batch to calls: invalid target address: illegal base64 data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			operationID, err := ton.OperationID(tt.batchOp, tt.action, tt.predecessor, tt.salt)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, operationID)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
