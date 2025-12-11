package sui

import (
	"context"
	"math/big"
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

	mockbindutils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mockmcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mocksui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
	"github.com/smartcontractkit/mcms/types"
)

// executorTestExecutingCallbackParams is a mock implementation of ExecutingCallbackAppender for testing
type executorTestExecutingCallbackParams struct {
	client        sui.ISuiAPI
	mcms          *mockmcms.IMcms
	mcmsPackageID string
	registryObj   string
	accountObj    string
}

func (t *executorTestExecutingCallbackParams) AppendPTB(ctx context.Context, ptb *transaction.Transaction, executeCallback *transaction.Argument, calls []Call) error {
	// For testing, just return success without actually building the complex PTB
	return nil
}

var accountObj = "0xaccount"
var mcmsObj = "0xmcms"
var registryObj = "0xregistry"
var timelockObj = "0xtimelock"

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	encoder := &Encoder{
		ChainSelector: 1,
		TxCount:       5,
	}

	mcmsPackageID := "0x123456789abcdef"
	role := TimelockRoleProposer

	entrypointEncoder := &MockEntrypointArgEncoder{t: t, registryObj: "0xregistry"}

	executor, err := NewExecutor(
		mockClient,
		mockSigner,
		encoder,
		entrypointEncoder,
		mcmsPackageID,
		role,
		mcmsObj,
		accountObj,
		registryObj,
		timelockObj,
	)

	require.NoError(t, err)
	assert.NotNil(t, executor)
	assert.Equal(t, encoder, executor.Encoder)
	assert.NotNil(t, executor.Inspector)
	assert.Equal(t, mockClient, executor.client)
	assert.Equal(t, mockSigner, executor.signer)
	assert.Equal(t, mcmsPackageID, executor.mcmsPackageID)
	assert.NotNil(t, executor.mcms)
	assert.Equal(t, mcmsObj, executor.mcmsObj)
	assert.Equal(t, accountObj, executor.accountObj)
	assert.Equal(t, registryObj, executor.registryObj)
	assert.Equal(t, timelockObj, executor.timelockObj)
}

func TestEncodeSignatures(t *testing.T) {
	t.Parallel()

	signatures := []types.Signature{
		{
			R: common.BigToHash(big.NewInt(0x1234)),
			S: common.BigToHash(big.NewInt(0x5678)),
			V: 27,
		},
		{
			R: common.BigToHash(big.NewInt(0x9abc)),
			S: common.BigToHash(big.NewInt(0xdef0)),
			V: 28,
		},
	}

	encoded := encodeSignatures(signatures)

	require.Len(t, encoded, 2)

	// Check first signature
	assert.Len(t, encoded[0], 65)             // 32 + 32 + 1
	assert.Equal(t, byte(27), encoded[0][64]) // V value

	// Check second signature
	assert.Len(t, encoded[1], 65)             // 32 + 32 + 1
	assert.Equal(t, byte(28), encoded[1][64]) // V value
}

func TestEncodeSignatures_VOffset(t *testing.T) {
	t.Parallel()

	signatures := []types.Signature{
		{
			R: common.BigToHash(big.NewInt(0x1234)),
			S: common.BigToHash(big.NewInt(0x5678)),
			V: 0, // Should be offset by 27
		},
		{
			R: common.BigToHash(big.NewInt(0x9abc)),
			S: common.BigToHash(big.NewInt(0xdef0)),
			V: 1, // Should be offset by 27
		},
	}

	encoded := encodeSignatures(signatures)

	require.Len(t, encoded, 2)

	// Check V offset is applied
	assert.Equal(t, byte(27), encoded[0][64]) // 0 + 27
	assert.Equal(t, byte(28), encoded[1][64]) // 1 + 27
}

func TestEncodeSignatures_EmptySignatures(t *testing.T) {
	t.Parallel()

	signatures := []types.Signature{}
	encoded := encodeSignatures(signatures)

	assert.Empty(t, encoded)
}

func TestEncodeSignatures_PaddingR(t *testing.T) {
	t.Parallel()

	// Test with a small R value that needs padding
	signatures := []types.Signature{
		{
			R: common.BigToHash(big.NewInt(0x1)),
			S: common.BigToHash(big.NewInt(0x2)),
			V: 27,
		},
	}

	encoded := encodeSignatures(signatures)

	require.Len(t, encoded, 1)
	assert.Len(t, encoded[0], 65) // 32 + 32 + 1

	// Check that the small values are properly padded
	// The R value (0x1) should be padded to 32 bytes and be at the end of the first 32 bytes
	assert.Equal(t, byte(0x1), encoded[0][31]) // Last byte of R section
	assert.Equal(t, byte(0x2), encoded[0][63]) // Last byte of S section
	assert.Equal(t, byte(27), encoded[0][64])  // V value
}

func TestEncodeSignatures_LargeVValue(t *testing.T) {
	t.Parallel()

	signatures := []types.Signature{
		{
			R: common.BigToHash(big.NewInt(0x1234)),
			S: common.BigToHash(big.NewInt(0x5678)),
			V: 30, // Large V value that shouldn't be offset
		},
	}

	encoded := encodeSignatures(signatures)

	require.Len(t, encoded, 1)
	// Check that large V values are not offset
	assert.Equal(t, byte(30), encoded[0][64]) // V value should remain unchanged
}

func TestExecutor_SetRoot_IfImplemented(t *testing.T) {
	t.Parallel()

	signatures := []types.Signature{
		{
			R: common.BigToHash(big.NewInt(0x1234)),
			S: common.BigToHash(big.NewInt(0x5678)),
			V: 27,
		},
		{
			R: common.BigToHash(big.NewInt(0x9abc)),
			S: common.BigToHash(big.NewInt(0xdef0)),
			V: 28,
		},
	}

	encoded := encodeSignatures(signatures)

	// Verify the signatures are properly encoded for SetRoot
	require.Len(t, encoded, 2)
	assert.Len(t, encoded[0], 65) // 32 + 32 + 1
	assert.Len(t, encoded[1], 65) // 32 + 32 + 1
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockSigner := mockbindutils.NewSuiSigner(t)
	mockedMcms := mockmcms.NewIMcms(t)

	executor := &Executor{
		signer:        mockSigner,
		mcms:          mockedMcms,
		mcmsPackageID: "0x123456789abcdef",
		mcmsObj:       mcmsObj,
		Encoder: &Encoder{
			ChainSelector:        types.ChainSelector(cselectors.SUI_TESTNET.Selector),
			TxCount:              5,
			OverridePreviousRoot: false,
		},
	}

	// Test data for SetRoot
	root := [32]byte{1, 2, 3, 4, 5}
	validUntil := uint32(1234567890)
	signatures := []types.Signature{
		{
			R: common.BigToHash(big.NewInt(0x1234)),
			S: common.BigToHash(big.NewInt(0x5678)),
			V: 27,
		},
	}

	metadata := types.ChainMetadata{
		StartingOpCount:  1,
		AdditionalFields: []byte(`{"role":2}`), // TimelockRole = 2
	}

	proof := []common.Hash{common.HexToHash("0x1234")}

	expectedResponse := &models.SuiTransactionBlockResponse{
		Digest: "0xsetroot123",
	}

	// Mock the SetRoot call
	mockedMcms.EXPECT().SetRoot(
		ctx,
		mock.AnythingOfType("*bind.CallOpts"),
		mock.AnythingOfType("bind.Object"), // stateObj
		mock.AnythingOfType("bind.Object"), // clockObj
		byte(2),                            // TimelockRole = 2
		root[:],
		uint64(validUntil),
		mock.AnythingOfType("*big.Int"), // chainId
		executor.mcmsPackageID,
		uint64(1), // startingOpCount
		uint64(6), // startingOpCount + txCount (1 + 5)
		false,     // overridePreviousRoot
		[][]byte{common.HexToHash("0x1234").Bytes()}, // metadataProof
		encodeSignatures(signatures),
	).Return(expectedResponse, nil)

	result, err := executor.SetRoot(ctx, metadata, proof, root, validUntil, signatures)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResponse.Digest, result.Hash)
	assert.Equal(t, expectedResponse, result.RawData)
}

func TestExecutor_ExecuteOperation_InvalidAdditionalFields(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)
	mockmcmsContract := mockmcms.NewIMcms(t)

	executor := &Executor{
		signer:        mockSigner,
		mcms:          mockmcmsContract,
		mcmsPackageID: "0x123456789abcdef",
		mcmsObj:       mcmsObj,
		client:        mockClient,
		Encoder: &Encoder{
			ChainSelector: types.ChainSelector(cselectors.SUI_TESTNET.Selector),
			TxCount:       5,
		},
	}

	// Test data with invalid additional fields (should fail early)
	metadata := types.ChainMetadata{
		AdditionalFields: []byte(`{"role":2}`),
	}
	nonce := uint32(123)
	proof := []common.Hash{common.HexToHash("0x1234")}
	op := types.Operation{
		Transaction: types.Transaction{
			To:               "0x123",
			Data:             []byte("test_data"),
			AdditionalFields: []byte("invalid_json"), // Invalid JSON
		},
	}

	result, err := executor.ExecuteOperation(ctx, metadata, nonce, proof, op)

	// Should error due to invalid additional fields
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal additional fields")
	assert.Empty(t, result.Hash)
}

func TestExecutor_ExecuteOperation_Success_ScheduleBatch(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)
	mockmcmsContract := mockmcms.NewIMcms(t)
	mockEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	executor := &Executor{
		signer:        mockSigner,
		mcms:          mockmcmsContract,
		mcmsPackageID: "0x123456789abcdef",
		mcmsObj:       mcmsObj,
		timelockObj:   timelockObj,
		registryObj:   registryObj,
		accountObj:    accountObj,
		client:        mockClient,
		Encoder: &Encoder{
			ChainSelector: types.ChainSelector(cselectors.SUI_TESTNET.Selector),
			TxCount:       5,
		},
		// Mock ExecutePTB function directly in the struct
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
	}

	// Test data for schedule batch operation
	metadata := types.ChainMetadata{
		AdditionalFields: []byte(`{"role":2}`), // TimelockRoleProposer = 2
	}
	nonce := uint32(123)
	proof := []common.Hash{common.HexToHash("0x1234")}
	op := types.Operation{
		Transaction: types.Transaction{
			To:   "0x0000000000000000000000000000000000000000000000000000000000000123",
			Data: []byte("test_data"),
			AdditionalFields: []byte(`{
				"module_name": "test_module",
				"function": "timelock_schedule_batch"
			}`),
		},
	}

	// Mock expectations for ExecuteOperation flow
	mockmcmsContract.EXPECT().Encoder().Return(mockEncoder).Times(2) // Called twice in the method

	// Mock the first Execute encoder call
	mockEncoder.EXPECT().Execute(
		mock.AnythingOfType("bind.Object"), // stateObj
		mock.AnythingOfType("bind.Object"), // clockObj
		byte(2),                            // TimelockRoleProposer
		mock.AnythingOfType("*big.Int"),    // chainId
		executor.mcmsPackageID,
		uint64(nonce),
		"0000000000000000000000000000000000000000000000000000000000000123", // to address (without 0x prefix)
		"test_module",
		"timelock_schedule_batch",
		[]byte("test_data"),
		[][]byte{common.HexToHash("0x1234").Bytes()},
	).Return(nil, nil)

	// Mock the first Bound() call for AppendPTB
	mockmcmsContract.EXPECT().Bound().Return(mockBound).Times(2) // Called twice
	mockBound.EXPECT().AppendPTB(ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Times(2)

	// Mock the timelock schedule call
	mockEncoder.EXPECT().DispatchTimelockScheduleBatchWithArgs(executor.timelockObj, "0x6", mock.Anything).Return(nil, nil)

	result, err := executor.ExecuteOperation(ctx, metadata, nonce, proof, op)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "9WzSXdwbky8tNbH7juvyaui4QzMUYEjdCEKMrMgLhXHT", result.Hash)
}

func TestExecutor_ExecuteOperation_Success_Bypass(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)
	mockmcmsContract := mockmcms.NewIMcms(t)
	mockEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	// Create a test ExecutingCallbackAppender
	testExecutingCallbackParams := &executorTestExecutingCallbackParams{
		client:        mockClient,
		mcms:          mockmcmsContract,
		mcmsPackageID: "0x123456789abcdef",
		registryObj:   registryObj,
		accountObj:    accountObj,
	}

	executor := &Executor{
		signer:        mockSigner,
		mcms:          mockmcmsContract,
		mcmsPackageID: "0x123456789abcdef",
		mcmsObj:       mcmsObj,
		timelockObj:   timelockObj,
		registryObj:   registryObj,
		accountObj:    accountObj,
		client:        mockClient,
		Encoder: &Encoder{
			ChainSelector: types.ChainSelector(cselectors.SUI_TESTNET.Selector),
			TxCount:       5,
		},
		// Mock ExecutePTB function directly in the struct
		ExecutePTB: func(ctx context.Context, opts *bind.CallOpts, client sui.ISuiAPI, ptb *transaction.Transaction) (*models.SuiTransactionBlockResponse, error) {
			return &models.SuiTransactionBlockResponse{
				Digest: "0xbypass_success_digest",
				Transaction: models.SuiTransactionBlock{
					Data: models.SuiTransactionBlockData{
						Sender: "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
					},
				},
			}, nil
		},
		executingCallbackParams: testExecutingCallbackParams,
	}

	// Test data for bypass operation with valid serialized bypass batch data
	metadata := types.ChainMetadata{
		AdditionalFields: []byte(`{"role":1}`), // TimelockRoleBypasser = 1
	}
	nonce := uint32(456)
	proof := []common.Hash{common.HexToHash("0x5678")}

	// Create test bypass batch data using the same serialization function
	target := make([]byte, 32)
	copy(target[12:], []byte("0xabcdef1234567890"))
	targets := [][]byte{target}
	moduleNames := []string{"test_module"}
	functionNames := []string{"test_function"}
	datas := [][]byte{{0xde, 0xad, 0xbe, 0xef}}

	bypassBatchData, err := serializeTimelockBypasserExecuteBatch(targets, moduleNames, functionNames, datas)
	require.NoError(t, err)

	op := types.Operation{
		Transaction: types.Transaction{
			To:   "0x0000000000000000000000000000000000000000000000000000000000000456",
			Data: bypassBatchData,
			AdditionalFields: []byte(`{
				"module_name": "test_module",
				"function": "timelock_bypasser_execute_batch",
				"internal_state_objects": ["0xstate1"],
				"internal_type_args": [["0xType1"]]
			}`),
		},
	}

	// Mock expectations for ExecuteOperation flow
	mockmcmsContract.EXPECT().Encoder().Return(mockEncoder).Times(2) // Called twice in the method

	// Mock the first Execute encoder call
	mockEncoder.EXPECT().Execute(
		mock.AnythingOfType("bind.Object"), // stateObj
		mock.AnythingOfType("bind.Object"), // clockObj
		byte(1),                            // TimelockRoleBypasser
		mock.AnythingOfType("*big.Int"),    // chainId
		executor.mcmsPackageID,
		uint64(nonce),
		"0000000000000000000000000000000000000000000000000000000000000456", // to address (without 0x prefix)
		"test_module",
		"timelock_bypasser_execute_batch",
		bypassBatchData,
		[][]byte{common.HexToHash("0x5678").Bytes()},
	).Return(nil, nil)

	// Mock the bound contract calls for the bypass flow
	mockmcmsContract.EXPECT().Bound().Return(mockBound).Times(2) // Called twice for bypass flow

	// Mock the first AppendPTB call for the execute call
	mockBound.EXPECT().AppendPTB(ctx, mock.AnythingOfType("*bind.CallOpts"), mock.AnythingOfType("*transaction.Transaction"), mock.Anything).Return(&transaction.Argument{}, nil).Once()

	// Mock the bypass timelock call
	mockEncoder.EXPECT().DispatchTimelockBypasserExecuteBatchWithArgs(mock.AnythingOfType("*transaction.Argument"), mock.AnythingOfType("string")).Return(nil, nil)

	// Mock the second AppendPTB call for the timelock call - return a mock executeCallback
	executeCallback := &transaction.Argument{}
	mockBound.EXPECT().AppendPTB(ctx, mock.AnythingOfType("*bind.CallOpts"), mock.AnythingOfType("*transaction.Transaction"), mock.Anything).Return(executeCallback, nil).Once()

	result, err := executor.ExecuteOperation(ctx, metadata, nonce, proof, op)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0xbypass_success_digest", result.Hash)
}

func TestExecutor_ExecuteOperation_Success_Cancel(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)
	mockmcmsContract := mockmcms.NewIMcms(t)
	mockEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	executor := &Executor{
		signer:        mockSigner,
		mcms:          mockmcmsContract,
		mcmsPackageID: "0x123456789abcdef",
		mcmsObj:       mcmsObj,
		timelockObj:   timelockObj,
		registryObj:   registryObj,
		accountObj:    accountObj,
		client:        mockClient,
		Encoder: &Encoder{
			ChainSelector: types.ChainSelector(cselectors.SUI_TESTNET.Selector),
			TxCount:       5,
		},
		// Mock ExecutePTB function directly in the struct
		ExecutePTB: func(ctx context.Context, opts *bind.CallOpts, client sui.ISuiAPI, ptb *transaction.Transaction) (*models.SuiTransactionBlockResponse, error) {
			return &models.SuiTransactionBlockResponse{
				Digest: "0xcancel_success_digest",
				Transaction: models.SuiTransactionBlock{
					Data: models.SuiTransactionBlockData{
						Sender: "0x742d35cc6b8d4c8c8e1b9b3b2d2a8b9c8d7e6f1234567890abcdef0123456789",
					},
				},
			}, nil
		},
	}

	// Test data for cancel operation
	// Create a mock operation ID that represents a previously scheduled operation
	operationID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}

	// Serialize the operation ID for cancellation (using the same function as the actual code)
	cancelData, err := serializeTimelockCancel(operationID)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		AdditionalFields: []byte(`{"role":3}`), // TimelockRoleCanceller = 3
	}
	nonce := uint32(789)
	proof := []common.Hash{common.HexToHash("0x9abc")}
	op := types.Operation{
		Transaction: types.Transaction{
			To:   "0x0000000000000000000000000000000000000000000000000000000000000789",
			Data: cancelData,
			AdditionalFields: []byte(`{
				"module_name": "mcms",
				"function": "timelock_cancel"
			}`),
		},
	}

	// Mock expectations for ExecuteOperation flow
	mockmcmsContract.EXPECT().Encoder().Return(mockEncoder).Times(2) // Called twice in the method

	// Mock the first Execute encoder call
	mockEncoder.EXPECT().Execute(
		mock.AnythingOfType("bind.Object"), // stateObj
		mock.AnythingOfType("bind.Object"), // clockObj
		byte(3),                            // TimelockRoleCanceller
		mock.AnythingOfType("*big.Int"),    // chainId
		executor.mcmsPackageID,
		uint64(nonce),
		"0000000000000000000000000000000000000000000000000000000000000789", // to address (without 0x prefix)
		"mcms",
		"timelock_cancel",
		cancelData,
		[][]byte{common.HexToHash("0x9abc").Bytes()},
	).Return(nil, nil)

	// Mock the Bound() call for AppendPTB
	mockmcmsContract.EXPECT().Bound().Return(mockBound).Times(2) // Called twice

	// Mock the first AppendPTB call for the execute call
	mockBound.EXPECT().AppendPTB(ctx, mock.AnythingOfType("*bind.CallOpts"), mock.AnythingOfType("*transaction.Transaction"), mock.Anything).Return(&transaction.Argument{}, nil).Once()

	// Mock the timelock cancel call
	mockEncoder.EXPECT().DispatchTimelockCancelWithArgs(executor.timelockObj, mock.Anything).Return(nil, nil)

	// Mock the second AppendPTB call for the timelock call
	mockBound.EXPECT().AppendPTB(ctx, mock.AnythingOfType("*bind.CallOpts"), mock.AnythingOfType("*transaction.Transaction"), mock.Anything).Return(nil, nil).Once()

	result, err := executor.ExecuteOperation(ctx, metadata, nonce, proof, op)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0xcancel_success_digest", result.Hash)
}

func TestExecutor_ExecuteOperation_Cancel_InvalidOperationID(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)
	mockmcmsContract := mockmcms.NewIMcms(t)

	executor := &Executor{
		signer:        mockSigner,
		mcms:          mockmcmsContract,
		mcmsPackageID: "0x123456789abcdef",
		mcmsObj:       mcmsObj,
		timelockObj:   timelockObj,
		registryObj:   registryObj,
		accountObj:    accountObj,
		client:        mockClient,
		Encoder: &Encoder{
			ChainSelector: types.ChainSelector(cselectors.SUI_TESTNET.Selector),
			TxCount:       5,
		},
	}

	// Test data with invalid operation ID (too short)
	invalidOperationID := []byte{0x01, 0x02} // Only 2 bytes instead of 32
	cancelData, err := serializeTimelockCancel(invalidOperationID)
	require.NoError(t, err)

	metadata := types.ChainMetadata{
		AdditionalFields: []byte(`{"role":3}`), // TimelockRoleCanceller = 3
	}
	nonce := uint32(999)
	proof := []common.Hash{common.HexToHash("0xdef0")}
	op := types.Operation{
		Transaction: types.Transaction{
			To:   "0x0000000000000000000000000000000000000000000000000000000000000999",
			Data: cancelData,
			AdditionalFields: []byte(`{
				"module_name": "mcms",
				"function": "timelock_cancel"
			}`),
		},
	}

	// The operation should still succeed as the validation happens at the contract level
	// This test ensures our executor handles various operation ID sizes correctly
	mockEncoder := mockmcms.NewMcmsEncoder(t)
	mockBound := mockbindutils.NewIBoundContract(t)

	executor.ExecutePTB = func(ctx context.Context, opts *bind.CallOpts, client sui.ISuiAPI, ptb *transaction.Transaction) (*models.SuiTransactionBlockResponse, error) {
		return &models.SuiTransactionBlockResponse{
			Digest: "0xcancel_invalid_op_digest",
		}, nil
	}

	mockmcmsContract.EXPECT().Encoder().Return(mockEncoder).Times(2)
	mockmcmsContract.EXPECT().Bound().Return(mockBound).Times(2)

	mockEncoder.EXPECT().Execute(
		mock.AnythingOfType("bind.Object"),
		mock.AnythingOfType("bind.Object"),
		byte(3),
		mock.AnythingOfType("*big.Int"),
		executor.mcmsPackageID,
		uint64(nonce),
		"0000000000000000000000000000000000000000000000000000000000000999",
		"mcms",
		"timelock_cancel",
		cancelData,
		[][]byte{common.HexToHash("0xdef0").Bytes()},
	).Return(nil, nil)

	mockBound.EXPECT().AppendPTB(ctx, mock.AnythingOfType("*bind.CallOpts"), mock.AnythingOfType("*transaction.Transaction"), mock.Anything).Return(&transaction.Argument{}, nil).Once()
	mockEncoder.EXPECT().DispatchTimelockCancelWithArgs(executor.timelockObj, mock.Anything).Return(nil, nil)
	mockBound.EXPECT().AppendPTB(ctx, mock.AnythingOfType("*bind.CallOpts"), mock.AnythingOfType("*transaction.Transaction"), mock.Anything).Return(nil, nil).Once()

	result, err := executor.ExecuteOperation(ctx, metadata, nonce, proof, op)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0xcancel_invalid_op_digest", result.Hash)
}
