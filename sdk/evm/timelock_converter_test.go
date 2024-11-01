package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverterEVM_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	timelockAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	zeroHash := common.Hash{}

	testCases := []struct {
		name           string
		txn            types.BatchChainOperation
		minDelay       string
		operation      types.TimelockAction
		predecessor    common.Hash
		expectedError  error
		expectedOpType string
	}{
		{
			name: "Schedule operation",
			txn: types.BatchChainOperation{
				Batch: []types.Operation{
					NewEVMOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      types.TimelockActionSchedule,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			txn: types.BatchChainOperation{
				Batch: []types.Operation{
					NewEVMOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      types.TimelockActionCancel,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			txn: types.BatchChainOperation{
				Batch: []types.Operation{
					NewEVMOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      types.TimelockAction("invalid"),
			predecessor:    zeroHash,
			expectedError:  &core.InvalidTimelockOperationError{ReceivedTimelockOperation: "invalid"},
			expectedOpType: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			converter := &TimelockConverterEVM{}
			chainOperation, operationId, err := converter.ConvertBatchToChainOperation(tc.txn, timelockAddress, tc.minDelay, tc.operation, tc.predecessor)

			if tc.expectedError != nil {
				require.Error(t, err)
				assert.IsType(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, common.Hash{}, operationId)
				assert.Equal(t, tc.expectedOpType, chainOperation.Operation.ContractType)
				assert.Equal(t, timelockAddress.Hex(), chainOperation.Operation.To)
				assert.Equal(t, tc.txn.ChainSelector, chainOperation.ChainSelector)
			}
		})
	}
}
