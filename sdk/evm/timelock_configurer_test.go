package evm

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_mocks "github.com/smartcontractkit/mcms/sdk/evm/mocks"
)

const (
	testDefaultGasLimit  = uint64(8000000)
	testDefaultBalance   = 1e18
	testSimulatedChainID = 1337
)

type testSimulatedChain struct {
	backend *simulated.Backend
	signers []*testSigner
}

type testSigner struct {
	privateKey *ecdsa.PrivateKey
}

func newTestSimulatedChain(t *testing.T, numSigners uint64) testSimulatedChain {
	t.Helper()

	signers := make([]*testSigner, 0, numSigners)
	for range numSigners {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)

		signers = append(signers, &testSigner{privateKey: key})
	}

	genesisAlloc := evmTypes.GenesisAlloc{}
	for _, signer := range signers {
		genesisAlloc[signer.address(t)] = evmTypes.Account{
			Balance: big.NewInt(testDefaultBalance),
		}
	}

	return testSimulatedChain{
		backend: simulated.NewBackend(
			genesisAlloc,
			simulated.WithBlockGasLimit(testDefaultGasLimit),
		),
		signers: signers,
	}
}

func (s *testSigner) newTransactOpts(t *testing.T) *bind.TransactOpts {
	t.Helper()

	auth, err := bind.NewKeyedTransactorWithChainID(s.privateKey, big.NewInt(testSimulatedChainID))
	require.NoError(t, err)
	auth.GasLimit = testDefaultGasLimit

	return auth
}

func (s *testSigner) address(t *testing.T) common.Address {
	t.Helper()

	publicKeyECDSA, ok := s.privateKey.Public().(*ecdsa.PublicKey)
	require.True(t, ok)

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

func (s testSimulatedChain) deployRBACTimelock(
	t *testing.T,
	signer *testSigner,
	admin common.Address,
) *bindings.RBACTimelock {
	t.Helper()

	_, _, timelock, err := bindings.DeployRBACTimelock(
		signer.newTransactOpts(t),
		s.backend.Client(),
		big.NewInt(0),
		admin,
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	s.backend.Commit()

	return timelock
}

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

	_, err := configurer.GrantRole(t.Context(), "not-an-address", sdk.TimelockRoleProposer, validAddress)
	require.ErrorContains(t, err, "invalid timelock address")

	_, err = configurer.GrantRole(t.Context(), common.Address{}.Hex(), sdk.TimelockRoleProposer, validAddress)
	require.ErrorContains(t, err, "invalid timelock address")

	_, err = configurer.GrantRole(t.Context(), validTimelock, sdk.TimelockRole(99), validAddress)
	require.ErrorContains(t, err, "invalid timelock role")

	_, err = configurer.GrantRole(t.Context(), validTimelock, sdk.TimelockRoleProposer, "not-an-address")
	require.ErrorContains(t, err, "invalid target address")

	_, err = configurer.GrantRole(t.Context(), validTimelock, sdk.TimelockRoleProposer, common.Address{}.Hex())
	require.ErrorContains(t, err, "invalid target address")
}

func TestTimelockConfigurer_GrantRoleNoSend(t *testing.T) {
	t.Parallel()

	sim := newTestSimulatedChain(t, 3)
	timelock := sim.deployRBACTimelock(t, sim.signers[0], sim.signers[0].address(t))

	auth := sim.signers[0].newTransactOpts(t)
	auth.NoSend = true
	configurer := NewTimelockConfigurer(sim.backend.Client(), auth)

	role := sdk.TimelockRoleProposer
	roleHash, err := role.Hash()
	require.NoError(t, err)
	target := sim.signers[1].address(t)

	result, err := configurer.GrantRole(t.Context(), timelock.Address().Hex(), role, target.Hex())
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

	sim := newTestSimulatedChain(t, 2)
	timelock := sim.deployRBACTimelock(t, sim.signers[0], sim.signers[0].address(t))

	configurer := NewTimelockConfigurer(sim.backend.Client(), sim.signers[0].newTransactOpts(t))
	role := sdk.TimelockRoleExecutor
	roleHash, err := role.Hash()
	require.NoError(t, err)
	target := sim.signers[1].address(t)

	result, err := configurer.GrantRole(t.Context(), timelock.Address().Hex(), role, target.Hex())
	require.NoError(t, err)
	require.NotEmpty(t, result.Hash)
	require.Equal(t, chainsel.FamilyEVM, result.ChainFamily)
	tx, ok := result.RawData.(*evmTypes.Transaction)
	require.True(t, ok)
	require.NotNil(t, tx)

	sim.backend.Commit()

	hasRole, err := timelock.HasRole(&bind.CallOpts{Context: t.Context()}, [32]byte(roleHash), target)
	require.NoError(t, err)
	require.True(t, hasRole)
}
