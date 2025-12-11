package evm

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/ethereum/go-ethereum/common"

	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConverter_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	timelockAddress := "0x1234567890123456789012345678901234567890"
	mcmAddress := "0x9876543210987654321098765432109876543210"
	zeroHash := common.Hash{}
	testCases := []struct {
		name           string
		metadata       types.ChainMetadata
		op             types.BatchOperation
		delay          string
		operation      types.TimelockAction
		predecessor    common.Hash
		salt           common.Hash
		expectedError  string
		expectedOpType string
	}{
		{
			name: "Schedule operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{
					NewTransaction(
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
			salt:           zeroHash,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{
					NewTransaction(
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
			salt:           zeroHash,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			op: types.BatchOperation{
				Transactions: []types.Transaction{
					NewTransaction(
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
			salt:           zeroHash,
			expectedError:  sdkerrors.NewInvalidTimelockOperationError("invalid").Error(),
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
				ChainSelector: types.ChainSelector(cselectors.ETHEREUM_TESTNET_SEPOLIA.Selector),
			},
			delay:         "1h",
			operation:     types.TimelockActionSchedule,
			predecessor:   zeroHash,
			salt:          zeroHash,
			expectedError: "failed to unmarshal EVM additional fields: invalid character 'i' looking for beginning of value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			converter := &TimelockConverter{}
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

			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.NotEqual(t, common.Hash{}, operationID)
				require.Len(t, chainOperations, 1)
				require.Equal(t, timelockAddress, chainOperations[0].Transaction.To)
				require.Equal(t, tc.op.ChainSelector, chainOperations[0].ChainSelector)
			}
		})
	}
}
