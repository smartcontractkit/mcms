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
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk/evm"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewSimulator(t *testing.T) {
	t.Parallel()

	mockEncoder := &evm.Encoder{}
	mockClient := evm_mocks.NewContractDeployBackend(t)

	// Test error cases
	tests := []struct {
		name       string
		encoder    *evm.Encoder
		client     evm.ContractDeployBackend
		wantErr    error
		wantErrMsg string
	}{
		{
			name:       "success",
			encoder:    mockEncoder,
			client:     mockClient,
			wantErr:    nil,
			wantErrMsg: "",
		},
		{
			name:       "nil encoder",
			encoder:    nil,
			client:     mockClient,
			wantErr:    errors.New("Simulator was created without an encoder"),
			wantErrMsg: "Simulator was created without an encoder",
		},
		{
			name:       "nil client",
			encoder:    mockEncoder,
			client:     nil,
			wantErr:    errors.New("Simulator was created without an inspector"),
			wantErrMsg: "Simulator was created without an inspector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			simulator, err := evm.NewSimulator(tt.encoder, tt.client)
			if tt.wantErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, simulator, "expected non-nil simulator when no error is returned")
			} else {
				assert.Nil(t, simulator, "expected nil simulator when error is returned")
				require.EqualError(t, err, tt.wantErrMsg, "expected error message to match")
				assert.Equal(t, tt.wantErr, err, "expected error to match")
			}
		})
	}
}

func TestSimulator_ExecuteOperation(t *testing.T) {
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
				From:    common.HexToAddress("0xFrom"),
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
				m.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, nil)
			},
			wantTxHash: "0xc381f411283719726be93f957b9e3ca7d8041725c22fefab8dcf132770adf7a9",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)

			if tt.mockSetup != nil {
				tt.mockSetup(client)
			}

			simulator, err := evm.NewSimulator(tt.encoder, client)
			require.NoError(t, err)

			err = simulator.SimulateOperation(
				context.TODO(),
				tt.metadata,
				tt.op,
			)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimulator_SetRoot(t *testing.T) {
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
				From:    common.HexToAddress("0xFrom"),
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
				m.EXPECT().CallContract(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, nil)
			},
			wantTxHash: "0xc381f411283719726be93f957b9e3ca7d8041725c22fefab8dcf132770adf7a9",
			wantErr:    nil,
		},
		{
			name: "failure in geth operation conversion due to invalid chain ID",
			auth: &bind.TransactOpts{
				From:    common.HexToAddress("0xFrom"),
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

			simulator, err := evm.NewSimulator(tt.encoder, client)
			require.NoError(t, err)
			err = simulator.SimulateSetRoot(
				context.TODO(),
				tt.auth.From.Hex(),
				tt.metadata,
				tt.proof,
				tt.root,
				tt.validUntil,
				tt.sortedSignatures)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
