package evm

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
)

func TestNewTimelockConfigurer(t *testing.T) {
	t.Parallel()

	mockClient := evm_mocks.NewContractDeployBackend(t)
	mockAuth := &bind.TransactOpts{}

	configurer := NewTimelockConfigurer(mockClient, mockAuth)

	assert.Equal(t, mockClient, configurer.client)
	assert.Equal(t, mockAuth, configurer.auth)
}

func TestTimelockConfigurer_UpdateDelay(t *testing.T) {
	t.Parallel()

	mockAuth := &bind.TransactOpts{
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

	sharedMockSetup := func(m *evm_mocks.ContractDeployBackend) {
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
	}

	tests := []struct {
		name            string
		timelockAddress string
		newDelay        uint64
		mockSetup       func(m *evm_mocks.ContractDeployBackend)
		wantErr         bool
	}{
		{
			name:            "success",
			timelockAddress: "0xMockedTimelockAddress",
			newDelay:        120,
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(nil)
				sharedMockSetup(m)
			},
			wantErr: false,
		},
		{
			name:            "failure in tx execution",
			timelockAddress: "0xMockedTimelockAddress",
			newDelay:        120,
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(errors.New("error during tx send"))
				sharedMockSetup(m)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := evm_mocks.NewContractDeployBackend(t)
			if test.mockSetup != nil {
				test.mockSetup(client)
			}

			configurer := NewTimelockConfigurer(client, mockAuth)
			result, err := configurer.UpdateDelay(t.Context(), test.timelockAddress, test.newDelay)

			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.Hash)
			}
		})
	}
}
