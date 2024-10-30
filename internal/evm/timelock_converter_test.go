package evm

import (
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/core/proposal/timelock"

	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimelockConverterEVM_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	timelockAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	zeroHash := common.Hash{}

	testCases := []struct {
		name           string
		txn            timelock.BatchChainOperation
		minDelay       string
		operation      timelock.TimelockOperationType
		predecessor    common.Hash
		expectedError  error
		expectedOpType string
	}{
		{
			name: "Schedule operation",
			txn: timelock.BatchChainOperation{
				Batch: []mcms.Operation{
					NewEVMOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      timelock.Schedule,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			txn: timelock.BatchChainOperation{
				Batch: []mcms.Operation{
					NewEVMOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      timelock.Cancel,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			txn: timelock.BatchChainOperation{
				Batch: []mcms.Operation{
					NewEVMOperation(
						common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						[]byte("data"),
						big.NewInt(1000),
						"RBACTimelock",
						[]string{"tag1", "tag2"},
					),
				},
				ChainSelector: mcms.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      timelock.TimelockOperationType("invalid"),
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
