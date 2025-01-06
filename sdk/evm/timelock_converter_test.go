package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverter_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	timelockAddress := "0x1234567890123456789012345678901234567890"
	zeroHash := common.Hash{}

	testCases := []struct {
		name           string
		op             types.BatchOperation
		delay          string
		operation      types.TimelockAction
		predecessor    common.Hash
		expectedError  error
		expectedOpType string
	}{
		{
			name: "Schedule operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{
					NewOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			delay:          "1h",
			operation:      types.TimelockActionSchedule,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{
					NewOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			delay:          "1h",
			operation:      types.TimelockActionCancel,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{
					NewOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			delay:          "1h",
			operation:      types.TimelockAction("invalid"),
			predecessor:    zeroHash,
			expectedError:  sdkerrors.NewInvalidTimelockOperationError("invalid"),
			expectedOpType: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			converter := &TimelockConverter{}
			chainOperation, operationId, err := converter.ConvertBatchToChainOperation(
				tc.op, NewEVMContractID(timelockAddress), types.MustParseDuration(tc.delay), tc.operation, tc.predecessor,
			)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.IsType(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, common.Hash{}, operationId)
				assert.Equal(t, tc.expectedOpType, chainOperation.Transaction.ContractType)
				assert.Equal(t, timelockAddress, chainOperation.Transaction.To)
				assert.Equal(t, tc.op.ChainSelector, chainOperation.ChainSelector)
			}
		})
	}
}
