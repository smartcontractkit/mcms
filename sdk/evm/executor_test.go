package evm_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
	"github.com/smartcontractkit/mcms/types"
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

func TestExecutor_ExecuteOperation(t *testing.T) {
	t.Parallel()

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
				m.On("SendTransaction", mock.Anything, mock.Anything).
					Return(nil)
				m.On("HeaderByNumber", mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)
				m.On("SuggestGasPrice", mock.Anything).
					Return(big.NewInt(100000000), nil)
				m.On("PendingCodeAt", mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)
				m.On("EstimateGas", mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
				m.On("PendingNonceAt", mock.Anything, mock.Anything).
					Return(uint64(1), nil)
			},
			wantTxHash: "0xc381f411283719726be93f957b9e3ca7d8041725c22fefab8dcf132770adf7a9",
			wantErr:    nil,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)

			if tt.mockSetup != nil {
				tt.mockSetup(client)
			}

			executor := evm.NewExecutor(tt.encoder, client, tt.auth)
			txHash, err := executor.ExecuteOperation(tt.metadata, tt.nonce, tt.proof, tt.op)

			assert.Equal(t, tt.wantTxHash, txHash)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

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
				m.On("SendTransaction", mock.Anything, mock.Anything).
					Return(nil)
				m.On("HeaderByNumber", mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)
				m.On("SuggestGasPrice", mock.Anything).
					Return(big.NewInt(100000000), nil)
				m.On("PendingCodeAt", mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)
				m.On("EstimateGas", mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
				m.On("PendingNonceAt", mock.Anything, mock.Anything).
					Return(uint64(1), nil)
			},
			wantTxHash: "0xc381f411283719726be93f957b9e3ca7d8041725c22fefab8dcf132770adf7a9",
			wantErr:    nil,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)

			if tt.mockSetup != nil {
				tt.mockSetup(client)
			}

			executor := evm.NewExecutor(tt.encoder, client, tt.auth)
			txHash, err := executor.SetRoot(tt.metadata,
				tt.proof,
				tt.root,
				tt.validUntil,
				tt.sortedSignatures)

			assert.Equal(t, tt.wantTxHash, txHash)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
