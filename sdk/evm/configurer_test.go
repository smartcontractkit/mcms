package evm_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/mcms/sdk/evm"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
	"github.com/smartcontractkit/mcms/types"
)

// TestConfigurer_SetConfig tests the SetConfig method of the Configurer.
func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	// Helper function to create a common.Address from string
	addr := func(address string) common.Address {
		return common.HexToAddress(address)
	}

	tests := []struct {
		name       string
		mcmAddr    string
		auth       *bind.TransactOpts
		cfg        *types.Config
		clearRoot  bool
		mockSetup  func(m *evm_mocks.ContractDeployBackend)
		wantTxHash string
		wantErr    error
	}{
		{
			name:    "success",
			mcmAddr: "0xMockedMCMAddress",
			auth: &bind.TransactOpts{
				From: addr("0xMockedFromAddress"),
				Signer: func(address common.Address, tx *evmTypes.Transaction) (*evmTypes.Transaction, error) {
					return tx, nil
				},
			},
			cfg: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					addr("0xSigner1"),
					addr("0xSigner2"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							addr("0xGroupSigner1"),
						},
						GroupSigners: nil,
					},
				},
			},
			clearRoot: true,
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Mock SuggestGasPrice
				m.EXPECT().SuggestGasPrice(mock.Anything).
					Return(big.NewInt(100000000), nil)

				// Mock PendingNonceAt
				m.EXPECT().PendingNonceAt(mock.Anything, addr("0xMockedFromAddress")).
					Return(uint64(1), nil)

				// Mock SendTransaction
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(nil)

				// Mock HeaderByNumber (if used internally)
				m.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)

				// Mock PendingCodeAt (if used internally)
				m.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)

				// Mock EstimateGas (if used internally)
				m.EXPECT().EstimateGas(mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
			},
			wantTxHash: "0x861a7de18a94850d8af57088385267ebd680a6397ad5be37bf0851371b051942",
			wantErr:    nil,
		},
		{
			name:    "failure - SendTransaction fails",
			mcmAddr: "0xMockedMCMAddress",
			auth: &bind.TransactOpts{
				From: addr("0xMockedFromAddress"),
				Signer: func(address common.Address, tx *evmTypes.Transaction) (*evmTypes.Transaction, error) {
					return tx, nil
				},
			},
			cfg: &types.Config{
				Quorum: 1,
				Signers: []common.Address{
					addr("0xSigner1"),
				},
				GroupSigners: nil,
			},
			clearRoot: false,
			mockSetup: func(m *evm_mocks.ContractDeployBackend) {
				// Mock SuggestGasPrice
				m.EXPECT().SuggestGasPrice(mock.Anything).
					Return(big.NewInt(100000000), nil)

				// Mock PendingNonceAt
				m.EXPECT().PendingNonceAt(mock.Anything, addr("0xMockedFromAddress")).
					Return(uint64(1), nil)

				// Mock SendTransaction to return an error
				m.EXPECT().SendTransaction(mock.Anything, mock.Anything).
					Return(errors.New("transaction failed"))

				// Mock HeaderByNumber (if used internally)
				m.EXPECT().HeaderByNumber(mock.Anything, mock.Anything).
					Return(&evmTypes.Header{}, nil)

				// Mock PendingCodeAt (if used internally)
				m.EXPECT().PendingCodeAt(mock.Anything, mock.Anything).
					Return([]byte("0x01"), nil)

				// Mock EstimateGas (if used internally)
				m.EXPECT().EstimateGas(mock.Anything, mock.Anything).
					Return(uint64(50000), nil)
			},
			wantTxHash: "",
			wantErr:    errors.New("transaction failed"),
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Initialize the mock ContractDeployBackend
			client := evm_mocks.NewContractDeployBackend(t)

			// Apply the mock setup for the ContractDeployBackend
			if tt.mockSetup != nil {
				tt.mockSetup(client)
			}

			// Create the Configurer instance
			configurer := evm.NewConfigurer(client, tt.auth)

			// Call SetConfig
			txHash, err := configurer.SetConfig(tt.mcmAddr, tt.cfg, tt.clearRoot)

			// Assert the results
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				assert.Equal(t, "", txHash)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTxHash, txHash)
			}

			// Assert that all expectations were met
			client.AssertExpectations(t)
		})
	}
}
