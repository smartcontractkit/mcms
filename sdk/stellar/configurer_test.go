package stellar_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/sdk/stellar"
	stellarmocks "github.com/smartcontractkit/mcms/sdk/stellar/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	mcmAddr := testContractID(t, 50)
	signer := common.HexToAddress("0x1234")

	invoker := stellarmocks.NewInvoker(t)
	invoker.EXPECT().
		InvokeContract(
			mock.Anything,
			mcmAddr,
			"set_config",
			mock.Anything,
		).
		Return(nil, nil).
		Once()

	configurer := stellar.NewConfigurer(invoker)

	cfg, err := types.NewConfig(1, []common.Address{signer}, nil)
	require.NoError(t, err)

	res, err := configurer.SetConfig(t.Context(), mcmAddr, &cfg, true)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, res.ChainFamily)
}

func TestConfigurer_SetConfig_InvalidInput(t *testing.T) {
	t.Parallel()

	validConfig := func(t *testing.T) types.Config {
		t.Helper()

		cfg, err := types.NewConfig(
			1,
			[]common.Address{common.HexToAddress("0x1234")},
			nil,
		)
		require.NoError(t, err)

		return cfg
	}

	tests := []struct {
		name    string
		mcmAddr func(*testing.T) string
		config  func(*testing.T) *types.Config
		wantErr string
	}{
		{
			name: "invalid contract ID",
			mcmAddr: func(*testing.T) string {
				return "not-a-stellar-contract"
			},
			config: func(t *testing.T) *types.Config {
				t.Helper()
				return new(validConfig(t))
			},
			wantErr: "invalid contract ID",
		},
		{
			name: "nil config",
			mcmAddr: func(t *testing.T) string {
				t.Helper()
				return testContractID(t, 51)
			},
			config: func(*testing.T) *types.Config {
				return nil
			},
			wantErr: "config is nil",
		},
		{
			name: "invalid config",
			mcmAddr: func(t *testing.T) string {
				t.Helper()
				return testContractID(t, 52)
			},
			config: func(*testing.T) *types.Config {
				cfg := types.Config{}
				return &cfg
			},
			wantErr: "invalid config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			invoker := stellarmocks.NewInvoker(t)
			configurer := stellar.NewConfigurer(invoker)

			_, err := configurer.SetConfig(
				t.Context(),
				tt.mcmAddr(t),
				tt.config(t),
				true,
			)

			require.ErrorContains(t, err, tt.wantErr)
			invoker.AssertNotCalled(
				t,
				"InvokeContract",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
		})
	}
}

func TestConfigurer_SetConfigInputs(t *testing.T) {
	t.Parallel()

	mcmAddr := testContractID(t, 53)
	cfg := testMCMSConfig(t)

	signerAddresses,
		signerGroups,
		groupQuorums,
		groupParents,
		err := stellar.ConfigToSetConfigInputs(&cfg)
	require.NoError(t, err)

	invoker := stellarmocks.NewInvoker(t)
	invoker.EXPECT().
		InvokeContract(
			mock.Anything,
			mcmAddr,
			"set_config",
			mock.Anything,
		).
		Return(nil, nil).
		Once()

	configurer := stellar.NewConfigurer(invoker)

	res, err := configurer.SetConfigInputs(
		t.Context(),
		mcmAddr,
		signerAddresses,
		signerGroups,
		groupQuorums,
		groupParents,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, res.ChainFamily)
}

func TestConfigurer_SetConfigInputs_InvalidContractID(t *testing.T) {
	t.Parallel()

	invoker := stellarmocks.NewInvoker(t)
	configurer := stellar.NewConfigurer(invoker)

	_, err := configurer.SetConfigInputs(
		t.Context(),
		"not-a-stellar-contract",
		mcmsbindings.SignerAddresses{},
		mcmsbindings.SignerGroups{},
		[32]byte{},
		[32]byte{},
		false,
	)

	require.ErrorContains(t, err, "invalid contract ID")
	invoker.AssertNotCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestConfigurer_Ownership(t *testing.T) {
	t.Parallel()

	mcmAddr := testContractID(t, 60)
	newOwner := testContractID(t, 61)

	invoker := stellarmocks.NewInvoker(t)
	invoker.EXPECT().
		InvokeContract(mock.Anything, mcmAddr, "transfer_ownership", mock.Anything).
		Return(nil, nil).
		Once()
	invoker.EXPECT().
		InvokeContract(mock.Anything, mcmAddr, "accept_ownership", mock.Anything).
		Return(nil, nil).
		Once()
	invoker.EXPECT().
		InvokeContract(mock.Anything, mcmAddr, "cancel_ownership_transfer", mock.Anything).
		Return(nil, nil).
		Once()

	configurer := stellar.NewConfigurer(invoker)

	res, err := configurer.TransferOwnership(t.Context(), mcmAddr, newOwner)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, res.ChainFamily)

	res, err = configurer.AcceptOwnership(t.Context(), mcmAddr)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, res.ChainFamily)

	res, err = configurer.CancelOwnershipTransfer(t.Context(), mcmAddr)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, res.ChainFamily)
}

func TestConfigurer_TransferOwnership_InvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mcmAddr  func(*testing.T) string
		newOwner func(*testing.T) string
		wantErr  string
	}{
		{
			name: "invalid contract ID",
			mcmAddr: func(*testing.T) string {
				return "not-a-stellar-contract"
			},
			newOwner: func(t *testing.T) string {
				t.Helper()
				return testContractID(t, 70)
			},
			wantErr: "invalid contract ID",
		},
		{
			name: "empty new owner",
			mcmAddr: func(t *testing.T) string {
				t.Helper()
				return testContractID(t, 71)
			},
			newOwner: func(*testing.T) string {
				return ""
			},
			wantErr: "new owner is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			invoker := stellarmocks.NewInvoker(t)
			configurer := stellar.NewConfigurer(invoker)

			_, err := configurer.TransferOwnership(
				t.Context(),
				tt.mcmAddr(t),
				tt.newOwner(t),
			)

			require.ErrorContains(t, err, tt.wantErr)
			invoker.AssertNotCalled(
				t,
				"InvokeContract",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
		})
	}
}

func TestConfigurer_AcceptOwnership_InvalidContractID(t *testing.T) {
	t.Parallel()

	invoker := stellarmocks.NewInvoker(t)
	configurer := stellar.NewConfigurer(invoker)

	_, err := configurer.AcceptOwnership(t.Context(), "not-a-stellar-contract")

	require.ErrorContains(t, err, "invalid contract ID")
	invoker.AssertNotCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestConfigurer_CancelOwnershipTransfer_InvalidContractID(t *testing.T) {
	t.Parallel()

	invoker := stellarmocks.NewInvoker(t)
	configurer := stellar.NewConfigurer(invoker)

	_, err := configurer.CancelOwnershipTransfer(t.Context(), "not-a-stellar-contract")

	require.ErrorContains(t, err, "invalid contract ID")
	invoker.AssertNotCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestConfigurer_NilInvoker(t *testing.T) {
	t.Parallel()

	configurer := stellar.NewConfigurer(nil)
	cfg := testMCMSConfig(t)

	_, err := configurer.SetConfig(
		t.Context(),
		testContractID(t, 80),
		&cfg,
		false,
	)

	require.ErrorContains(t, err, "invoker is nil")
}
