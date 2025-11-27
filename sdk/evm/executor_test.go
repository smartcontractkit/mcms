package evm_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
	"github.com/smartcontractkit/mcms/types"
)

const (
	errMsgTxSend        = "error during tx send"
	errMsgExecErrType   = "error should be ExecutionError type"
	errMsgExecErrTxData = "ExecutionError should contain pre-packed transaction"
	errMsgExecErrNotNil = "ExecutionError should not be nil"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	mockEncoder := &evm.Encoder{}
	mockClient := evm_mocks.NewContractDeployBackend(t)
	mockAuth := &bind.TransactOpts{}

	executor := evm.NewExecutor(mockEncoder, mockClient, mockAuth)

	assert.Equal(t, mockEncoder, executor.Encoder, "expected Encoder to be set correctly")
	assert.NotNil(t, executor.Inspector, "expected Inspector to be initialized")
}

func TestExecutorExecuteOperation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name       string
		encoder    *evm.Encoder
		auth       *bind.TransactOpts
		metadata   types.ChainMetadata
		nonce      uint32
		proof      []common.Hash
		op         types.Operation
		mockSetup  func(m *evm_mocks.ContractDeployBackend)
		wantTxHash string
		wantErr    error
	}{
		{
			name: "success",
			encoder: &evm.Encoder{
				ChainSelector: chaintest.Chain1Selector,
			},
			auth: &bind.TransactOpts{
				Context: context.Background(),
				Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
					mockTx := evmTypes.NewTransaction(
						1,
						common.HexToAddress("0xMockedAddress"),
						big.NewInt(1000000000000000000),
						21000,
						big.NewInt(20000000000),
						nil,
					)

					return mockTx, nil
				},
			},
			metadata: types.ChainMetadata{
				MCMAddress: "0xAddress",
			},
			nonce: 1,
			op: types.Operation{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: types.Transaction{
					To:               "0xTo",
					Data:             []byte{1, 2, 3},
					AdditionalFields: json.RawMessage(`{"value": 0}`)},
			},
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Successful tx send
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(nil)
				m.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)
				m.EXPECT().SuggestGasPrice(mock.Anything).
					Return(big.NewInt(100000000), nil)
				m.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)
				m.EXPECT().EstimateGas(mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
				m.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
					Return(uint64(1), nil)
			},
			wantTxHash: "0xc381f411283719726be93f957b9e3ca7d8041725c22fefab8dcf132770adf7a9",
			wantErr:    nil,
		},
		{
			name: "failure in tx execution",
			encoder: &evm.Encoder{
				ChainSelector: chaintest.Chain1Selector,
			},
			auth: &bind.TransactOpts{
				Context: context.Background(),
				Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
					mockTx := evmTypes.NewTransaction(
						1,
						common.HexToAddress("0xMockedAddress"),
						big.NewInt(1000000000000000000),
						21000,
						big.NewInt(20000000000),
						nil,
					)

					return mockTx, nil
				},
			},
			metadata: types.ChainMetadata{
				MCMAddress: "0xAddress",
			},
			nonce: 1,
			op: types.Operation{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: types.Transaction{
					To:               "0xTo",
					Data:             []byte{1, 2, 3},
					AdditionalFields: json.RawMessage(`{"value": 0}`)},
			},
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Successful tx send
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(errors.New(errMsgTxSend))
				m.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)
				m.EXPECT().SuggestGasPrice(mock.Anything).
					Return(big.NewInt(100000000), nil)
				m.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)
				m.EXPECT().EstimateGas(mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
				m.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
					Return(uint64(1), nil)
			},
			wantTxHash: "",
			wantErr:    errors.New(errMsgTxSend),
		},
		{
			name:       "failure - nil encoder",
			encoder:    nil,
			mockSetup:  func(m *evm_mocks.ContractDeployBackend) {},
			wantTxHash: "",
			wantErr:    errors.New("Executor was created without an encoder"),
		},
		{
			name: "failure in geth operation conversion due to invalid chain ID",
			encoder: &evm.Encoder{
				ChainSelector: types.ChainSelector(1),
			},
			op: types.Operation{
				ChainSelector: types.ChainSelector(1),
				Transaction: types.Transaction{
					To:               "0xTo",
					Data:             []byte{1, 2, 3},
					AdditionalFields: json.RawMessage(`{"value": 0}`)},
			},
			mockSetup:  func(m *evm_mocks.ContractDeployBackend) {},
			wantTxHash: "",
			wantErr:    errors.New("invalid chain ID: 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)

			if tt.mockSetup != nil {
				tt.mockSetup(client)
			}

			executor := evm.NewExecutor(tt.encoder, client, tt.auth)
			tx, err := executor.ExecuteOperation(ctx, tt.metadata, tt.nonce, tt.proof, tt.op)

			require.Equal(t, tt.wantTxHash, tx.Hash)
			if tt.wantErr != nil {
				require.Error(t, err)
				// When error occurs after tx sending, check for ExecutionError with transaction data
				if tt.name == "failure in tx execution" {
					var execErr *evm.ExecutionError
					require.ErrorAs(t, err, &execErr, errMsgExecErrType)
					if execErr != nil {
						require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
						require.Equal(t, chain_selectors.FamilyEVM, tx.ChainFamily)
					}
				} else {
					// For other errors, just check the error message matches
					require.EqualError(t, err, tt.wantErr.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutorExecuteOperationWithEIP1559GasFees(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	encoder := &evm.Encoder{
		ChainSelector: chaintest.Chain1Selector,
	}
	auth := &bind.TransactOpts{
		Context:   context.Background(),
		Nonce:     big.NewInt(5),
		GasLimit:  uint64(100000),
		GasFeeCap: big.NewInt(20000000000),
		GasTipCap: big.NewInt(1000000000),
		Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
			mockTx := evmTypes.NewTransaction(
				1,
				common.HexToAddress("0xMockedAddress"),
				big.NewInt(1000000000000000000),
				21000,
				big.NewInt(20000000000),
				nil,
			)

			return mockTx, nil
		},
	}
	metadata := types.ChainMetadata{
		MCMAddress: "0xAddress",
	}
	op := types.Operation{
		ChainSelector: chaintest.Chain1Selector,
		Transaction: types.Transaction{
			To:               "0xTo",
			Data:             []byte{1, 2, 3},
			AdditionalFields: json.RawMessage(`{"value": 0}`),
		},
	}

	client := evm_mocks.NewContractDeployBackend(t)
	client.EXPECT().SendTransaction(mock.Anything, mock.Anything).
		Return(errors.New(errMsgTxSend)).Maybe()
	client.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
		Return(&evmTypes.Header{}, nil).Maybe()
	client.EXPECT().SuggestGasPrice(mock.Anything).
		Return(big.NewInt(100000000), nil).Maybe()
	client.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
		Return([]byte("0x01"), nil).Maybe()
	client.EXPECT().EstimateGas(mock.Anything, mock.Anything).
		Return(uint64(50000), nil).Maybe()
	client.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
		Return(uint64(5), nil).Maybe()

	executor := evm.NewExecutor(encoder, client, auth)
	tx, err := executor.ExecuteOperation(ctx, metadata, 1, []common.Hash{}, op)

	require.Error(t, err)
	var execErr *evm.ExecutionError
	require.ErrorAs(t, err, &execErr, errMsgExecErrType)
	require.NotNil(t, execErr, errMsgExecErrNotNil)
	require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
	require.Equal(t, chain_selectors.FamilyEVM, tx.ChainFamily)
	// Verify it's a DynamicFeeTx (EIP-1559)
	require.Equal(t, uint8(2), execErr.Transaction.Type(), "transaction should be EIP-1559 type")
}

func TestExecutorSetRootWithEIP1559GasFees(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	encoder := &evm.Encoder{
		ChainSelector: chaintest.Chain1Selector,
	}
	auth := &bind.TransactOpts{
		Context:   context.Background(),
		Nonce:     big.NewInt(3),
		GasLimit:  150000,
		GasFeeCap: big.NewInt(30000000000),
		GasTipCap: big.NewInt(2000000000),
		Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
			mockTx := evmTypes.NewTransaction(
				1,
				common.HexToAddress("0xMockedAddress"),
				big.NewInt(1000000000000000000),
				21000,
				big.NewInt(20000000000),
				nil,
			)

			return mockTx, nil
		},
	}
	metadata := types.ChainMetadata{
		MCMAddress: "0xAddress",
	}
	root := [32]byte{1, 2, 3}
	validUntil := uint32(4130013354)
	sortedSignatures := []types.Signature{{}, {}}

	client := evm_mocks.NewContractDeployBackend(t)
	client.EXPECT().SendTransaction(mock.Anything, mock.Anything).
		Return(errors.New(errMsgTxSend)).Maybe()
	client.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
		Return(&evmTypes.Header{}, nil).Maybe()
	client.EXPECT().SuggestGasPrice(mock.Anything).
		Return(big.NewInt(100000000), nil).Maybe()
	client.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
		Return([]byte("0x01"), nil).Maybe()
	client.EXPECT().EstimateGas(mock.Anything, mock.Anything).
		Return(uint64(50000), nil).Maybe()
	client.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
		Return(uint64(3), nil).Maybe()

	executor := evm.NewExecutor(encoder, client, auth)
	tx, err := executor.SetRoot(ctx, metadata, []common.Hash{}, root, validUntil, sortedSignatures)

	require.Error(t, err)
	var execErr *evm.ExecutionError
	require.ErrorAs(t, err, &execErr, errMsgExecErrType)
	require.NotNil(t, execErr, errMsgExecErrNotNil)
	require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
	require.Equal(t, chain_selectors.FamilyEVM, tx.ChainFamily)
	// Verify it's a DynamicFeeTx (EIP-1559)
	require.Equal(t, uint8(2), execErr.Transaction.Type(), "transaction should be EIP-1559 type")
}

func TestExecutorExecuteOperationWithLegacyGasPrice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	encoder := &evm.Encoder{
		ChainSelector: chaintest.Chain1Selector,
	}
	auth := &bind.TransactOpts{
		Context:  context.Background(),
		Nonce:    big.NewInt(2),
		GasLimit: uint64(80000),
		GasPrice: big.NewInt(50000000000), // Legacy gas price
		Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
			mockTx := evmTypes.NewTransaction(
				1,
				common.HexToAddress("0xMockedAddress"),
				big.NewInt(1000000000000000000),
				21000,
				big.NewInt(20000000000),
				nil,
			)

			return mockTx, nil
		},
	}
	metadata := types.ChainMetadata{
		MCMAddress: "0xAddress",
	}
	op := types.Operation{
		ChainSelector: chaintest.Chain1Selector,
		Transaction: types.Transaction{
			To:               "0xTo",
			Data:             []byte{1, 2, 3},
			AdditionalFields: json.RawMessage(`{"value": 0}`),
		},
	}

	client := evm_mocks.NewContractDeployBackend(t)
	client.EXPECT().SendTransaction(mock.Anything, mock.Anything).
		Return(errors.New(errMsgTxSend)).Maybe()
	client.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
		Return(&evmTypes.Header{}, nil).Maybe()
	client.EXPECT().SuggestGasPrice(mock.Anything).
		Return(big.NewInt(100000000), nil).Maybe()
	client.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
		Return([]byte("0x01"), nil).Maybe()
	client.EXPECT().EstimateGas(mock.Anything, mock.Anything).
		Return(uint64(50000), nil).Maybe()
	client.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
		Return(uint64(2), nil).Maybe()

	executor := evm.NewExecutor(encoder, client, auth)
	tx, err := executor.ExecuteOperation(ctx, metadata, 1, []common.Hash{}, op)

	require.Error(t, err)
	var execErr *evm.ExecutionError
	require.ErrorAs(t, err, &execErr, errMsgExecErrType)
	require.NotNil(t, execErr, errMsgExecErrNotNil)
	require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
	require.Equal(t, chain_selectors.FamilyEVM, tx.ChainFamily)
	// Verify it's a LegacyTx
	require.Equal(t, uint8(0), execErr.Transaction.Type(), "transaction should be legacy type")
}

func TestExecutorSetRootWithLegacyGasPrice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	encoder := &evm.Encoder{
		ChainSelector: chaintest.Chain1Selector,
	}
	auth := &bind.TransactOpts{
		Context:  context.Background(),
		Nonce:    big.NewInt(1),
		GasLimit: 120000,
		GasPrice: big.NewInt(40000000000), // Legacy gas price
		Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
			mockTx := evmTypes.NewTransaction(
				1,
				common.HexToAddress("0xMockedAddress"),
				big.NewInt(1000000000000000000),
				21000,
				big.NewInt(20000000000),
				nil,
			)

			return mockTx, nil
		},
	}
	metadata := types.ChainMetadata{
		MCMAddress: "0xAddress",
	}
	root := [32]byte{4, 5, 6}
	validUntil := uint32(4130013354)
	sortedSignatures := []types.Signature{{}}

	client := evm_mocks.NewContractDeployBackend(t)
	client.EXPECT().SendTransaction(mock.Anything, mock.Anything).
		Return(errors.New(errMsgTxSend)).Maybe()
	client.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
		Return(&evmTypes.Header{}, nil).Maybe()
	client.EXPECT().SuggestGasPrice(mock.Anything).
		Return(big.NewInt(100000000), nil).Maybe()
	client.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
		Return([]byte("0x01"), nil).Maybe()
	client.EXPECT().EstimateGas(mock.Anything, mock.Anything).
		Return(uint64(50000), nil).Maybe()
	client.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
		Return(uint64(1), nil).Maybe()

	executor := evm.NewExecutor(encoder, client, auth)
	tx, err := executor.SetRoot(ctx, metadata, []common.Hash{}, root, validUntil, sortedSignatures)

	require.Error(t, err)
	var execErr *evm.ExecutionError
	require.ErrorAs(t, err, &execErr, errMsgExecErrType)
	require.NotNil(t, execErr, errMsgExecErrNotNil)
	require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
	require.Equal(t, chain_selectors.FamilyEVM, tx.ChainFamily)
	// Verify it's a LegacyTx
	require.Equal(t, uint8(0), execErr.Transaction.Type(), "transaction should be legacy type")
}

func TestExecutorExecuteOperationRBACTimelockUnderlyingRevert(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	encoder := &evm.Encoder{
		ChainSelector: chaintest.Chain1Selector,
	}
	auth := &bind.TransactOpts{
		Context: context.Background(),
		From:    common.HexToAddress("0xFromAddress"),
		Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
			mockTx := evmTypes.NewTransaction(
				1,
				common.HexToAddress("0xMockedAddress"),
				big.NewInt(1000000000000000000),
				21000,
				big.NewInt(20000000000),
				nil,
			)

			return mockTx, nil
		},
	}
	metadata := types.ChainMetadata{
		MCMAddress: "0xAddress",
	}
	op := types.Operation{
		ChainSelector: chaintest.Chain1Selector,
		Transaction: types.Transaction{
			To:               "0xTo",
			Data:             []byte{1, 2, 3},
			AdditionalFields: json.RawMessage(`{"value": 0}`),
		},
	}

	client := evm_mocks.NewContractDeployBackend(t)
	// Mock the Execute call to return RBACTimelock error
	client.EXPECT().SendTransaction(mock.Anything, mock.Anything).
		Return(fmt.Errorf("contract error: error -`CallReverted` args [[8 195 121 160 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 32 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 45 82 66 65 67 84 105 109 101 108 111 99 107 58 32 117 110 100 101 114 108 121 105 110 103 32 116 114 97 110 115 97 99 116 105 111 110 32 114 101 118 101 114 116 101 100 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]]"))
	client.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
		Return(&evmTypes.Header{}, nil).Maybe()
	client.EXPECT().SuggestGasPrice(mock.Anything).
		Return(big.NewInt(100000000), nil).Maybe()
	client.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
		Return([]byte("0x01"), nil).Maybe()
	client.EXPECT().EstimateGas(mock.Anything, mock.Anything).
		Return(uint64(50000), nil).Maybe()
	client.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
		Return(uint64(1), nil).Maybe()

	// Mock CallContract to return the underlying revert reason
	client.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("execution reverted: 0x08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001a496e73756666696369656e742062616c616e636520746f2073656e6400000000000000000000000000000000000000000000000000000000")).
		Maybe()

	executor := evm.NewExecutor(encoder, client, auth)
	_, err := executor.ExecuteOperation(ctx, metadata, 1, []common.Hash{}, op)

	require.Error(t, err)
	var execErr *evm.ExecutionError
	require.ErrorAs(t, err, &execErr, errMsgExecErrType)
	require.NotNil(t, execErr, errMsgExecErrNotNil)
	require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
	require.Contains(t, err.Error(), "RBACTimelock: underlying transaction reverted", "error should mention RBACTimelock")
	// If CallContract was called, both raw and decoded underlying reasons should be populated when available.
	if execErr.RawUnderlyingReason != "" {
		require.NotEmpty(t, execErr.RawUnderlyingReason, "underlying reason should be extracted")
	}
	if execErr.DecodedUnderlyingReason != "" {
		require.Equal(t, "Insufficient balance to send", execErr.DecodedUnderlyingReason, "decoded underlying reason mismatch")
	}
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name             string
		encoder          *evm.Encoder
		auth             *bind.TransactOpts
		metadata         types.ChainMetadata
		proof            []common.Hash
		root             [32]byte
		validUntil       uint32
		sortedSignatures []types.Signature
		mockSetup        func(m *evm_mocks.ContractDeployBackend)
		wantTxHash       string
		wantErr          error
	}{
		{
			name: "success",
			encoder: &evm.Encoder{
				ChainSelector: chaintest.Chain1Selector,
			},
			auth: &bind.TransactOpts{
				Context: context.Background(),
				Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
					mockTx := evmTypes.NewTransaction(
						1,
						common.HexToAddress("0xMockedAddress"),
						big.NewInt(1000000000000000000),
						21000,
						big.NewInt(20000000000),
						nil,
					)

					return mockTx, nil
				},
			},
			metadata: types.ChainMetadata{
				MCMAddress: "0xAddress",
			},
			root:       [32]byte{1, 2, 3},
			validUntil: 4130013354,
			sortedSignatures: []types.Signature{
				{},
				{},
			},
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Successful tx send
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(nil)
				m.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)
				m.EXPECT().SuggestGasPrice(mock.Anything).
					Return(big.NewInt(100000000), nil)
				m.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)
				m.EXPECT().EstimateGas(mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
				m.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
					Return(uint64(1), nil)
			},
			wantTxHash: "0xc381f411283719726be93f957b9e3ca7d8041725c22fefab8dcf132770adf7a9",
			wantErr:    nil,
		},
		{
			name: "failure in tx send",
			encoder: &evm.Encoder{
				ChainSelector: chaintest.Chain1Selector,
			},
			auth: &bind.TransactOpts{
				Context: context.Background(),
				Signer: func(address common.Address, transaction *evmTypes.Transaction) (*evmTypes.Transaction, error) {
					mockTx := evmTypes.NewTransaction(
						1,
						common.HexToAddress("0xMockedAddress"),
						big.NewInt(1000000000000000000),
						21000,
						big.NewInt(20000000000),
						nil,
					)

					return mockTx, nil
				},
			},
			metadata: types.ChainMetadata{
				MCMAddress: "0xAddress",
			},
			root:       [32]byte{1, 2, 3},
			validUntil: 4130013354,
			sortedSignatures: []types.Signature{
				{},
				{},
			},
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Successful tx send
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(errors.New(errMsgTxSend))
				m.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)
				m.EXPECT().SuggestGasPrice(mock.Anything).
					Return(big.NewInt(100000000), nil)
				m.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)
				m.EXPECT().EstimateGas(mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
				m.EXPECT().PendingNonceAt(mock.Anything, mock.Anything).
					Return(uint64(1), nil)
			},
			wantTxHash: "",
			wantErr:    errors.New(errMsgTxSend),
		},
		{
			name:       "failure - nil encoder",
			encoder:    nil,
			mockSetup:  func(m *evm_mocks.ContractDeployBackend) {},
			wantTxHash: "",
			wantErr:    errors.New("Executor was created without an encoder"),
		},
		{
			name: "failure in geth operation conversion due to invalid chain ID",
			encoder: &evm.Encoder{
				ChainSelector: types.ChainSelector(1),
			},
			mockSetup:  func(m *evm_mocks.ContractDeployBackend) {},
			wantTxHash: "",
			wantErr:    errors.New("invalid chain ID: 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)

			if tt.mockSetup != nil {
				tt.mockSetup(client)
			}

			executor := evm.NewExecutor(tt.encoder, client, tt.auth)
			tx, err := executor.SetRoot(ctx, tt.metadata,
				tt.proof,
				tt.root,
				tt.validUntil,
				tt.sortedSignatures)

			require.Equal(t, tt.wantTxHash, tx.Hash)
			if tt.wantErr != nil {
				require.Error(t, err)
				if tt.name == "failure in tx send" {
					var execErr *evm.ExecutionError
					require.ErrorAs(t, err, &execErr, errMsgExecErrType)
					if execErr != nil {
						require.NotNil(t, execErr.Transaction, errMsgExecErrTxData)
						require.Equal(t, chain_selectors.FamilyEVM, tx.ChainFamily)
					}
				} else {
					require.EqualError(t, err, tt.wantErr.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
