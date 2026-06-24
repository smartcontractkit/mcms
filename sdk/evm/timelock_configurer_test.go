package evm

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
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
		wantErr         string
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
			wantErr: "failed to update delay",
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

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.Hash)
			}
		})
	}
}

func TestTimelockConfigurer_GrantRolesRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	configurer := NewTimelockConfigurer(evm_mocks.NewContractDeployBackend(t), &bind.TransactOpts{})
	validTimelock := common.HexToAddress("0x1000000000000000000000000000000000000001").Hex()
	validRole := sdk.TimelockRoleProposer
	validAddress := common.HexToAddress("0x2000000000000000000000000000000000000002").Hex()

	tests := []struct {
		name            string
		timelockAddress string
		role            sdk.TimelockRole
		addresses       []string
		wantErr         string
	}{
		{
			name:            "invalid timelock address",
			timelockAddress: "not-an-address",
			role:            validRole,
			addresses:       []string{validAddress},
			wantErr:         "invalid timelock address",
		},
		{
			name:            "invalid role",
			timelockAddress: validTimelock,
			role:            sdk.TimelockRoleUnknown,
			addresses:       []string{validAddress},
			wantErr:         "invalid timelock role",
		},
		{
			name:            "empty address list",
			timelockAddress: validTimelock,
			role:            validRole,
			addresses:       nil,
			wantErr:         "addresses must be non-empty",
		},
		{
			name:            "invalid target address",
			timelockAddress: validTimelock,
			role:            validRole,
			addresses:       []string{"not-an-address"},
			wantErr:         "invalid target address",
		},
		{
			name:            "zero target address",
			timelockAddress: validTimelock,
			role:            validRole,
			addresses:       []string{common.Address{}.Hex()},
			wantErr:         "invalid target address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := configurer.GrantRoles(context.Background(), tt.timelockAddress, tt.role, tt.addresses)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTimelockConfigurer_GrantRolesNoSend(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 3)
	timelock, _ := sim.DeployRBACTimelock(
		t,
		sim.Signers[0],
		sim.Signers[0].Address(t),
		nil,
		nil,
		nil,
		nil,
	)

	auth := sim.Signers[0].NewTransactOpts(t)
	auth.NoSend = true
	configurer := NewTimelockConfigurer(sim.Backend.Client(), auth)

	role := sdk.TimelockRoleProposer
	roleHash, err := role.Hash()
	require.NoError(t, err)
	targets := []common.Address{
		sim.Signers[1].Address(t),
		sim.Signers[2].Address(t),
	}
	targetStrings := []string{targets[0].Hex(), targets[1].Hex()}

	result, err := configurer.GrantRoles(t.Context(), timelock.Address().Hex(), role, targetStrings)
	require.NoError(t, err)

	txs, ok := result.RawData.([]*evmTypes.Transaction)
	require.True(t, ok)
	require.Len(t, txs, len(targets))

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRole := timelockABI.Methods["grantRole"]

	for i, tx := range txs {
		require.Equal(t, timelock.Address(), *tx.To())
		require.NotEmpty(t, tx.Data())
		require.Equal(t, grantRole.ID, tx.Data()[:4])

		decoded, err := grantRole.Inputs.Unpack(tx.Data()[4:])
		require.NoError(t, err)
		require.Len(t, decoded, 2)
		require.Equal(t, [32]byte(roleHash), decoded[0])
		require.Equal(t, targets[i], decoded[1])

		hasRole, err := timelock.HasRole(&bind.CallOpts{Context: t.Context()}, [32]byte(roleHash), targets[i])
		require.NoError(t, err)
		require.False(t, hasRole)
	}
}

func TestTimelockConfigurer_GrantRolesDirectSend(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 2)
	timelock, _ := sim.DeployRBACTimelock(
		t,
		sim.Signers[0],
		sim.Signers[0].Address(t),
		nil,
		nil,
		nil,
		nil,
	)

	configurer := NewTimelockConfigurer(sim.Backend.Client(), sim.Signers[0].NewTransactOpts(t))
	role := sdk.TimelockRoleExecutor
	roleHash, err := role.Hash()
	require.NoError(t, err)
	target := sim.Signers[1].Address(t)

	result, err := configurer.GrantRoles(t.Context(), timelock.Address().Hex(), role, []string{target.Hex()})
	require.NoError(t, err)
	require.NotEmpty(t, result.Hash)
	txs, ok := result.RawData.([]*evmTypes.Transaction)
	require.True(t, ok)
	require.Len(t, txs, 1)

	sim.Backend.Commit()

	hasRole, err := timelock.HasRole(&bind.CallOpts{Context: t.Context()}, [32]byte(roleHash), target)
	require.NoError(t, err)
	require.True(t, hasRole)
}
