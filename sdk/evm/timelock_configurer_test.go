package evm

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
)

const (
	timelockRoleUnknown  = 0
	timelockRoleExecutor = 4
	timelockRoleProposer = 5
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

func TestTimelockConfigurer_GrantRoleRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	configurer := NewTimelockConfigurer(evm_mocks.NewContractDeployBackend(t), &bind.TransactOpts{})
	validTimelock := common.HexToAddress("0x1000000000000000000000000000000000000001").Hex()
	validAddress := common.HexToAddress("0x2000000000000000000000000000000000000002").Hex()

	_, err := configurer.GrantRole(context.Background(), "not-an-address", timelockRoleProposer, validAddress)
	require.ErrorContains(t, err, "invalid timelock address")

	_, err = configurer.GrantRole(context.Background(), validTimelock, timelockRoleUnknown, validAddress)
	require.ErrorContains(t, err, "invalid timelock role")

	_, err = configurer.GrantRole(context.Background(), validTimelock, timelockRoleProposer, "not-an-address")
	require.ErrorContains(t, err, "invalid target address")

	_, err = configurer.GrantRole(context.Background(), validTimelock, timelockRoleProposer, common.Address{}.Hex())
	require.ErrorContains(t, err, "invalid target address")
}

func TestTimelockConfigurer_GrantRoleNoSend(t *testing.T) {
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

	roleHash := crypto.Keccak256Hash([]byte("PROPOSER_ROLE"))
	target := sim.Signers[1].Address(t)

	result, err := configurer.GrantRole(t.Context(), timelock.Address().Hex(), timelockRoleProposer, target.Hex())
	require.NoError(t, err)
	require.NotEmpty(t, result.Hash)
	require.Equal(t, chainsel.FamilyEVM, result.ChainFamily)

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRole := timelockABI.Methods["grantRole"]

	tx, ok := result.RawData.(*evmTypes.Transaction)
	require.True(t, ok)
	require.Equal(t, timelock.Address(), *tx.To())
	require.NotEmpty(t, tx.Data())
	require.GreaterOrEqual(t, len(tx.Data()), 4)
	require.Equal(t, grantRole.ID, tx.Data()[:4])

	decoded, err := grantRole.Inputs.Unpack(tx.Data()[4:])
	require.NoError(t, err)
	require.Len(t, decoded, 2)
	require.Equal(t, [32]byte(roleHash), decoded[0])
	require.Equal(t, target, decoded[1])

	hasRole, err := timelock.HasRole(&bind.CallOpts{Context: t.Context()}, [32]byte(roleHash), target)
	require.NoError(t, err)
	require.False(t, hasRole)
}

func TestTimelockConfigurer_GrantRoleDirectSend(t *testing.T) {
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
	roleHash := crypto.Keccak256Hash([]byte("EXECUTOR_ROLE"))
	target := sim.Signers[1].Address(t)

	result, err := configurer.GrantRole(t.Context(), timelock.Address().Hex(), timelockRoleExecutor, target.Hex())
	require.NoError(t, err)
	require.NotEmpty(t, result.Hash)
	require.Equal(t, chainsel.FamilyEVM, result.ChainFamily)
	tx, ok := result.RawData.(*evmTypes.Transaction)
	require.True(t, ok)
	require.NotNil(t, tx)

	sim.Backend.Commit()

	hasRole, err := timelock.HasRole(&bind.CallOpts{Context: t.Context()}, [32]byte(roleHash), target)
	require.NoError(t, err)
	require.True(t, hasRole)
}
