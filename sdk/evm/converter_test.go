package evm

import (
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
	timelockTypes "github.com/smartcontractkit/mcms/pkg/proposal/timelock/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestTimelockConverterEVM_ConvertBatchToChainOperation(t *testing.T) {
	timelockAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	zeroHash := common.Hash{}

	testCases := []struct {
		name           string
		txn            timelockTypes.BatchChainOperation
		minDelay       string
		operation      timelockTypes.TimelockOperationType
		predecessor    common.Hash
		expectedError  error
		expectedOpType string
	}{
		{
			name: "Schedule operation",
			txn: timelockTypes.BatchChainOperation{
				Batch: []types.Operation{
					{
						To:    common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						Data:  []byte("data"),
						Value: big.NewInt(1000),
						Tags:  []string{"tag1", "tag2"},
					},
				},
				ChainIdentifier: types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      timelockTypes.Schedule,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			txn: timelockTypes.BatchChainOperation{
				Batch: []types.Operation{
					{
						To:    common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						Data:  []byte("data"),
						Value: big.NewInt(1000),
						Tags:  []string{"tag1", "tag2"},
					},
				},
				ChainIdentifier: types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      timelockTypes.Cancel,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			txn: timelockTypes.BatchChainOperation{
				Batch: []types.Operation{
					{
						To:    common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
						Data:  []byte("data"),
						Value: big.NewInt(1000),
						Tags:  []string{"tag1", "tag2"},
					},
				},
				ChainIdentifier: types.ChainIdentifier(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			minDelay:       "1h",
			operation:      timelockTypes.TimelockOperationType("invalid"),
			predecessor:    zeroHash,
			expectedError:  &errors.InvalidTimelockOperationError{ReceivedTimelockOperation: "invalid"},
			expectedOpType: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			converter := &TimelockConverterEVM{}
			chainOperation, operationId, err := converter.ConvertBatchToChainOperation(tc.txn, timelockAddress, tc.minDelay, tc.operation, tc.predecessor)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.IsType(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, common.Hash{}, operationId)
				assert.Equal(t, tc.expectedOpType, chainOperation.Operation.ContractType)
				assert.Equal(t, timelockAddress, chainOperation.Operation.To)
				assert.Equal(t, tc.txn.ChainIdentifier, chainOperation.ChainIdentifier)
			}
		})
	}
}
