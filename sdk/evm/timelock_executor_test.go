package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockExecutor(t *testing.T) {
	t.Parallel()

	mockClient := evm_mocks.NewContractDeployBackend(t)
	mockAuth := &bind.TransactOpts{}

	executor := NewTimelockExecutor(mockClient, mockAuth)

	assert.Equal(t, mockClient, executor.client)
	assert.Equal(t, mockAuth, executor.auth)
}

func TestTimelockExecutor_Execute(t *testing.T) {
	t.Parallel()

	mockAuth := &bind.TransactOpts{
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
	}

	tests := []struct {
		name            string
		auth            *bind.TransactOpts
		timelockAddress string
		bop             types.BatchOperation
		predecessor     common.Hash
		salt            common.Hash
		mockSetup       func(m *evm_mocks.ContractDeployBackend)
		wantTxHash      string
		wantErr         error
	}{
		{
			name:            "success",
			auth:            mockAuth,
			timelockAddress: "0xMockedTimelockAddress",
			bop: types.BatchOperation{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{
					{
						To:               "0xTo",
						Data:             []byte{1, 2, 3},
						AdditionalFields: json.RawMessage(`{"value": 0}`)},
				},
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
			name:            "failure in tx execution",
			auth:            mockAuth,
			timelockAddress: "0xMockedTimelockAddress",
			bop: types.BatchOperation{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{
					{
						To:               "0xTo",
						Data:             []byte{1, 2, 3},
						AdditionalFields: json.RawMessage(`{"value": 0}`)},
				},
			},
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Successful tx send
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(fmt.Errorf("error during tx send"))
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
			wantErr:    fmt.Errorf("error during tx send"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)
			if test.mockSetup != nil {
				test.mockSetup(client)
			}

			executor := NewTimelockExecutor(client, test.auth)
			txHash, err := executor.Execute(test.bop, test.timelockAddress, test.predecessor, test.salt)

			assert.Equal(t, test.wantTxHash, txHash)
			if test.wantErr != nil {
				assert.EqualError(t, err, test.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
