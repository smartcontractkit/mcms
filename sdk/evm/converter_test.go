package evm

/**
func TestTimelockConverterEVM_ConvertBatchToChainOperation(t *testing.T) {
	t.Parallel()

	timelockAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	zeroHash := common.Hash{}

	testCases := []struct {
		name           string
		txn            timelock.BatchChainOperation
		minDelay       string
		operation      timelock.TimelockOperation
		predecessor    common.Hash
		expectedError  error
		expectedOpType string
	}{
		{
			name: "Schedule operation",
			txn: timelock.BatchChainOperation{
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
			operation:      timelock.Schedule,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Cancel operation",
			txn: timelock.BatchChainOperation{
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
			operation:      timelock.Cancel,
			predecessor:    zeroHash,
			expectedError:  nil,
			expectedOpType: "RBACTimelock",
		},
		{
			name: "Invalid operation",
			txn: timelock.BatchChainOperation{
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
			operation:      timelock.TimelockOperation("invalid"),
			predecessor:    zeroHash,
			expectedError:  &errors.InvalidTimelockOperationError{ReceivedTimelockOperation: "invalid"},
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
				assert.Equal(t, timelockAddress, chainOperation.Operation.To)
				assert.Equal(t, tc.txn.ChainIdentifier, chainOperation.ChainIdentifier)
			}
		})
	}
}
*/
