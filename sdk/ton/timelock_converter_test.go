package ton

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverter_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	timelockAddress := "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"
	mcmAddress := "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"
	zeroHash := common.Hash{}
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
			name: "Schedule operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{must(NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
					new(big.Int).SetUint64(1000),
					"RBACTimelock",
					[]string{"tag1", "tag2"},
				))},
				ChainSelector: types.ChainSelector(cselectors.TON_TESTNET.Selector),
			},
			delay:          "1h",
			operation:      types.TimelockActionSchedule,
			predecessor:    zeroHash,
			salt:           zeroHash,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{must(NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
					new(big.Int).SetUint64(1000),
					"RBACTimelock",
					[]string{"tag1", "tag2"},
				))},
				ChainSelector: types.ChainSelector(cselectors.TON_TESTNET.Selector),
			},
			delay:          "1h",
			operation:      types.TimelockActionCancel,
			predecessor:    zeroHash,
			salt:           zeroHash,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{must(NewTransaction(
					address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"),
					cell.BeginCell().MustStoreBinarySnake([]byte("data")).ToSlice(),
					new(big.Int).SetUint64(1000),
					"RBACTimelock",
					[]string{"tag1", "tag2"},
				))},
				ChainSelector: types.ChainSelector(cselectors.TON_TESTNET.Selector),
			},
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
					OperationMetadata: types.OperationMetadata{ContractType: "RBACTimelock"},
					To:                timelockAddress,
					Data:              []byte("0x1234"),
					AdditionalFields:  []byte("invalid"),
				}},
				ChainSelector: types.ChainSelector(cselectors.TON_TESTNET.Selector),
			},
			delay:       "1h",
			operation:   types.TimelockActionSchedule,
			predecessor: zeroHash,
			salt:        zeroHash,
			wantErr:     "failed to unmarshal additional fields: invalid character 'i' looking for beginning of value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			converter := NewTimelockConverter()
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
			}
		})
	}
}
