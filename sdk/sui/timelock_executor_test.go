package sui

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	moduleMcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"

	mockBindUtils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mockMcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mockSui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockExecutor(t *testing.T) {
	t.Parallel()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	mcmsPackageID := "0x123456789abcdef"

	executor, err := NewTimelockExecutor(mockClient, mockSigner, mcmsPackageID, registryObj, accountObj)

	require.NoError(t, err)
	assert.NotNil(t, executor)
	assert.Equal(t, mockClient, executor.client)
	assert.Equal(t, mockSigner, executor.signer)
	assert.Equal(t, mcmsPackageID, executor.mcmsPackageID)
	assert.Equal(t, registryObj, executor.registryObj)
	assert.Equal(t, accountObj, executor.accountObj)
	assert.NotNil(t, executor.TimelockInspector)
}

func TestTimelockExecutor_Properties(t *testing.T) {
	t.Parallel()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)

	mcmsPackageID := "0x123456789abcdef"

	executor, err := NewTimelockExecutor(mockClient, mockSigner, mcmsPackageID, registryObj, accountObj)
	require.NoError(t, err)

	// Test that the TimelockExecutor has all required properties
	assert.Equal(t, mcmsPackageID, executor.mcmsPackageID)
	assert.Equal(t, registryObj, executor.registryObj)
	assert.Equal(t, accountObj, executor.accountObj)
	assert.NotNil(t, executor.client)
	assert.NotNil(t, executor.signer)

	// Test that TimelockInspector is properly embedded
	assert.NotNil(t, executor.TimelockInspector)
	assert.Equal(t, mockClient, executor.TimelockInspector.client)
	assert.Equal(t, mockSigner, executor.TimelockInspector.signer)
	assert.Equal(t, mcmsPackageID, executor.TimelockInspector.mcmsPackageID)

	// Test that dependency injection functions are properly initialized
	assert.NotNil(t, executor.ExecutePTB)
	assert.NotNil(t, executor.AppendPTBFromExecutingCallbackParams)
}

func TestTimelockExecutor_Execute_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)
	mockMcmsContract := mockMcms.NewIMcms(t)
	mockEncoder := mockMcms.NewMcmsEncoder(t)
	mockBound := mockBindUtils.NewIBoundContract(t)

	// Create executor with dependency injection
	executor := &TimelockExecutor{
		TimelockInspector: TimelockInspector{
			client:        mockClient,
			signer:        mockSigner,
			mcmsPackageID: "0x123456789abcdef",
			mcms:          mockMcmsContract,
		},
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		registryObj:   registryObj,
		accountObj:    accountObj,
		// Mock ExecutePTB function
		ExecutePTB: func(ctx context.Context, opts *bind.CallOpts, client sui.ISuiAPI, ptb *transaction.Transaction) (*models.SuiTransactionBlockResponse, error) {
			return &models.SuiTransactionBlockResponse{
				Digest: "9WzSXdwbky8tNbH7juvyaui4QzMUYEjdCEKMrMgLhXHT",
				Transaction: models.SuiTransactionBlock{
					Data: models.SuiTransactionBlockData{
						Sender: "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
					},
				},
			}, nil
		},
		// Mock AppendPTBFromExecutingCallbackParams function
		AppendPTBFromExecutingCallbackParams: func(ctx context.Context, client sui.ISuiAPI, mcms moduleMcms.IMcms, ptb *transaction.Transaction, mcmsPackageID string, executeCallback *transaction.Argument, calls []Call, registryObj string, accountObj string) error {
			return nil // Success
		},
	}

	// Test data
	bop := types.BatchOperation{
		Transactions: []types.Transaction{
			{
				To:   "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
				Data: []byte("test_data"),
				AdditionalFields: func() []byte {
					fields := AdditionalFields{
						ModuleName: "test_module",
						Function:   "test_function",
						StateObj:   "0xstate123",
					}
					data, err := json.Marshal(fields)
					if err != nil {
						return nil
					}

					return data
				}(),
			},
		},
	}
	predecessor := common.HexToHash("0x1234")
	salt := common.HexToHash("0x5678")

	// Mock the MCMS contract calls
	mockMcmsContract.On("Encoder").Return(mockEncoder)
	mockMcmsContract.On("Bound").Return(mockBound)

	// Mock TimelockExecuteBatch call
	timelockObject := bind.Object{Id: timelockObj}
	clockObj := bind.Object{Id: "0x6"}
	mockEncoder.On("TimelockExecuteBatch",
		timelockObject,
		clockObj,
		[]string{"0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789"},
		[]string{"test_module"},
		[]string{"test_function"},
		[][]byte{[]byte("test_data")},
		predecessor.Bytes(),
		salt.Bytes(),
	).Return(&bind.EncodedCall{}, nil)

	// Mock AppendPTB call
	mockBound.On("AppendPTB",
		ctx,
		mock.MatchedBy(func(opts *bind.CallOpts) bool {
			return opts.Signer == mockSigner && opts.WaitForExecution == true
		}),
		mock.AnythingOfType("*transaction.Transaction"),
		mock.AnythingOfType("*bind.EncodedCall"),
	).Return(&transaction.Argument{}, nil)

	// Execute the method
	result, err := executor.Execute(ctx, bop, timelockObj, predecessor, salt)

	// Verify the result
	require.NoError(t, err)
	assert.Equal(t, "9WzSXdwbky8tNbH7juvyaui4QzMUYEjdCEKMrMgLhXHT", result.Hash)
	assert.Equal(t, cselectors.FamilySui, result.ChainFamily)
	assert.NotNil(t, result.RawData)

	// Verify all mocks were called
	mockMcmsContract.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBound.AssertExpectations(t)
}

func TestTimelockExecutor_Execute_InvalidAdditionalFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)
	mockMcmsContract := mockMcms.NewIMcms(t)

	executor := &TimelockExecutor{
		TimelockInspector: TimelockInspector{
			client:        mockClient,
			signer:        mockSigner,
			mcmsPackageID: "0x123456789abcdef",
			mcms:          mockMcmsContract,
		},
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		registryObj:   registryObj,
		accountObj:    accountObj,
	}

	// Test data with invalid additional fields
	bop := types.BatchOperation{
		Transactions: []types.Transaction{
			{
				To:               "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
				Data:             []byte("test_data"),
				AdditionalFields: []byte("invalid_json"),
			},
		},
	}
	predecessor := common.HexToHash("0x1234")
	salt := common.HexToHash("0x5678")

	// Execute the method
	result, err := executor.Execute(ctx, bop, timelockObj, predecessor, salt)

	// Should error due to invalid additional fields
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal additional fields")
	assert.Empty(t, result.Hash)
}

func TestTimelockExecutor_Execute_InvalidTargetAddress(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)
	mockMcmsContract := mockMcms.NewIMcms(t)

	executor := &TimelockExecutor{
		TimelockInspector: TimelockInspector{
			client:        mockClient,
			signer:        mockSigner,
			mcmsPackageID: "0x123456789abcdef",
			mcms:          mockMcmsContract,
		},
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		registryObj:   registryObj,
		accountObj:    accountObj,
	}

	// Test data with invalid target address
	bop := types.BatchOperation{
		Transactions: []types.Transaction{
			{
				To:   "invalid_address",
				Data: []byte("test_data"),
				AdditionalFields: func() []byte {
					fields := AdditionalFields{
						ModuleName: "test_module",
						Function:   "test_function",
						StateObj:   "0xstate123",
					}
					data, err := json.Marshal(fields)
					if err != nil {
						return nil
					}

					return data
				}(),
			},
		},
	}
	predecessor := common.HexToHash("0x1234")
	salt := common.HexToHash("0x5678")

	// Execute the method
	result, err := executor.Execute(ctx, bop, timelockObj, predecessor, salt)

	// Should error due to invalid target address
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse target address")
	assert.Empty(t, result.Hash)
}

func TestTimelockExecutor_Execute_TimelockExecuteBatchFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)
	mockMcmsContract := mockMcms.NewIMcms(t)
	mockEncoder := mockMcms.NewMcmsEncoder(t)

	executor := &TimelockExecutor{
		TimelockInspector: TimelockInspector{
			client:        mockClient,
			signer:        mockSigner,
			mcmsPackageID: "0x123456789abcdef",
			mcms:          mockMcmsContract,
		},
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		registryObj:   registryObj,
		accountObj:    accountObj,
	}

	// Test data
	bop := types.BatchOperation{
		Transactions: []types.Transaction{
			{
				To:   "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
				Data: []byte("test_data"),
				AdditionalFields: func() []byte {
					fields := AdditionalFields{
						ModuleName: "test_module",
						Function:   "test_function",
						StateObj:   "0xstate123",
					}
					data, err := json.Marshal(fields)
					if err != nil {
						return nil
					}

					return data
				}(),
			},
		},
	}
	timelockObj := "0xtimelock123"
	predecessor := common.HexToHash("0x1234")
	salt := common.HexToHash("0x5678")

	// Mock the MCMS contract calls
	mockMcmsContract.On("Encoder").Return(mockEncoder)

	// Mock TimelockExecuteBatch call to fail
	timelockObject := bind.Object{Id: timelockObj}
	clockObj := bind.Object{Id: "0x6"}
	mockEncoder.On("TimelockExecuteBatch",
		timelockObject,
		clockObj,
		[]string{"0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789"},
		[]string{"test_module"},
		[]string{"test_function"},
		[][]byte{[]byte("test_data")},
		predecessor.Bytes(),
		salt.Bytes(),
	).Return(nil, assert.AnError)

	// Execute the method
	result, err := executor.Execute(ctx, bop, timelockObj, predecessor, salt)

	// Should error due to TimelockExecuteBatch failure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute batch")
	assert.Empty(t, result.Hash)

	// Verify mocks were called
	mockMcmsContract.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
}

func TestTimelockExecutor_Execute_AppendPTBFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := mockSui.NewISuiAPI(t)
	mockSigner := mockBindUtils.NewSuiSigner(t)
	mockMcmsContract := mockMcms.NewIMcms(t)
	mockEncoder := mockMcms.NewMcmsEncoder(t)
	mockBound := mockBindUtils.NewIBoundContract(t)

	executor := &TimelockExecutor{
		TimelockInspector: TimelockInspector{
			client:        mockClient,
			signer:        mockSigner,
			mcmsPackageID: "0x123456789abcdef",
			mcms:          mockMcmsContract,
		},
		client:        mockClient,
		signer:        mockSigner,
		mcmsPackageID: "0x123456789abcdef",
		registryObj:   registryObj,
		accountObj:    accountObj,
	}

	// Test data
	bop := types.BatchOperation{
		Transactions: []types.Transaction{
			{
				To:   "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
				Data: []byte("test_data"),
				AdditionalFields: func() []byte {
					fields := AdditionalFields{
						ModuleName: "test_module",
						Function:   "test_function",
						StateObj:   "0xstate123",
					}
					data, err := json.Marshal(fields)
					if err != nil {
						return nil
					}

					return data
				}(),
			},
		},
	}
	timelockObj := "0xtimelock123"
	predecessor := common.HexToHash("0x1234")
	salt := common.HexToHash("0x5678")

	// Mock the MCMS contract calls
	mockMcmsContract.On("Encoder").Return(mockEncoder)
	mockMcmsContract.On("Bound").Return(mockBound)

	// Mock TimelockExecuteBatch call to succeed
	timelockObject := bind.Object{Id: timelockObj}
	clockObj := bind.Object{Id: "0x6"}
	mockEncoder.On("TimelockExecuteBatch",
		timelockObject,
		clockObj,
		[]string{"0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789"},
		[]string{"test_module"},
		[]string{"test_function"},
		[][]byte{[]byte("test_data")},
		predecessor.Bytes(),
		salt.Bytes(),
	).Return(&bind.EncodedCall{}, nil)

	// Mock AppendPTB call to fail
	mockBound.On("AppendPTB",
		ctx,
		mock.MatchedBy(func(opts *bind.CallOpts) bool {
			return opts.Signer == mockSigner && opts.WaitForExecution == true
		}),
		mock.AnythingOfType("*transaction.Transaction"),
		mock.AnythingOfType("*bind.EncodedCall"),
	).Return(nil, assert.AnError)

	// Execute the method
	result, err := executor.Execute(ctx, bop, timelockObj, predecessor, salt)

	// Should error due to AppendPTB failure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "building PTB for execute call")
	assert.Empty(t, result.Hash)

	// Verify mocks were called
	mockMcmsContract.AssertExpectations(t)
	mockEncoder.AssertExpectations(t)
	mockBound.AssertExpectations(t)
}
