package stellar_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/network"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms/sdk/stellar"
	stellarmocks "github.com/smartcontractkit/mcms/sdk/stellar/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestDeployer_DeployMCMS(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	const wasmPath = "/tmp/mcms.wasm"

	salt := [32]byte{1, 2, 3, 4}
	contractID := testContractID(t, 1)

	dependency.On("DeployContract", mock.Anything, wasmPath, salt).Return(contractID, nil).Once()

	deployer := stellar.NewDeployer(dependency)

	got, err := deployer.DeployMCMS(t.Context(), wasmPath, salt)
	require.NoError(t, err)
	require.Equal(t, contractID, got)
}

func TestDeployer_DeployMCMS_DeployError(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	expectedErr := errors.New("deploy failed")

	dependency.On("DeployContract", mock.Anything, "/tmp/mcms.wasm", [32]byte{}).Return("", expectedErr).Once()

	deployer := stellar.NewDeployer(dependency)

	_, err := deployer.DeployMCMS(t.Context(), "/tmp/mcms.wasm", [32]byte{})

	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(t, err, "deploy stellar MCMS")
}

func TestDeployer_DeployMCMS_EmptyWASMPath(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	deployer := stellar.NewDeployer(dependency)

	_, err := deployer.DeployMCMS(t.Context(), "", [32]byte{})

	require.ErrorContains(t, err, "WASM path is empty")
	dependency.AssertNotCalled(t, "DeployContract", mock.Anything, mock.Anything, mock.Anything)
}

func TestDeployer_DeployMCMS_EmptyContractID(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	dependency.On("DeployContract", mock.Anything, "/tmp/mcms.wasm", [32]byte{}).Return("", nil).Once()

	deployer := stellar.NewDeployer(dependency)

	_, err := deployer.DeployMCMS(t.Context(), "/tmp/mcms.wasm", [32]byte{})

	require.ErrorContains(t, err, "empty contract ID")
}

func TestDeployer_DeployMCMS_NilDeployer(t *testing.T) {
	t.Parallel()

	deployer := stellar.NewDeployer(nil)

	_, err := deployer.DeployMCMS(t.Context(), "/tmp/mcms.wasm", [32]byte{})

	require.ErrorContains(t, err, "deployer is nil")
}

func TestDeployer_InitializeMCMS(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	chainID := chainselectors.STELLAR_LOCALNET.ChainID
	contractID := testContractID(t, 1)
	owner := testAccountAddress(t)
	cfg := testMCMSConfig(t)

	signerAddresses, signerGroups, groupQuorums, groupParents, err := stellar.ConfigToSetConfigInputs(&cfg)
	require.NoError(t, err)

	networkID := network.ID(chainselectors.STELLAR_LOCALNET.Passphrase)

	expectedArgs := []xdr.ScVal{
		scval.AddressToScVal(owner),
		scval.Bytes32ToScVal(networkID),
		scval.MustToScVal(signerAddresses.ToScVal()),
		scval.MustToScVal(signerGroups.ToScVal()),
		scval.Bytes32ToScVal(groupQuorums),
		scval.Bytes32ToScVal(groupParents),
		scval.SymbolToScVal("PROPOSER"),
	}

	dependency.On("InvokeContract", mock.Anything, contractID, "initialize", expectedArgs).Return((*xdr.ScVal)(nil), nil).Once()

	deployer := stellar.NewDeployer(dependency)

	result, err := deployer.InitializeMCMS(t.Context(), stellar.InitializeMCMSInput{
		ContractID:    contractID,
		Owner:         owner,
		ChainID:       chainID,
		Config:        &cfg,
		InstanceLabel: "PROPOSER",
	})
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, result.ChainFamily)
}

func TestDeployer_InitializeMCMS_InvokeError(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	expectedErr := errors.New("invoke failed")
	cfg := testMCMSConfig(t)

	dependency.On("InvokeContract", mock.Anything, mock.Anything, "initialize", mock.Anything).Return((*xdr.ScVal)(nil), expectedErr).Once()

	deployer := stellar.NewDeployer(dependency)

	_, err := deployer.InitializeMCMS(t.Context(), stellar.InitializeMCMSInput{
		ContractID:    testContractID(t, 1),
		Owner:         testAccountAddress(t),
		ChainID:       chainselectors.STELLAR_LOCALNET.ChainID,
		Config:        &cfg,
		InstanceLabel: "PROPOSER",
	})

	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(t, err, "initialize stellar MCMS")
}

func TestDeployer_InitializeMCMS_NilDeployer(t *testing.T) {
	t.Parallel()

	deployer := stellar.NewDeployer(nil)

	_, err := deployer.InitializeMCMS(t.Context(), stellar.InitializeMCMSInput{})

	require.ErrorContains(t, err, "deployer is nil")
}

func TestDeployer_InitializeMCMS_InvalidInput(t *testing.T) {
	t.Parallel()

	cfg := testMCMSConfig(t)

	valid := stellar.InitializeMCMSInput{
		ContractID:    testContractID(t, 1),
		Owner:         testAccountAddress(t),
		ChainID:       chainselectors.STELLAR_LOCALNET.ChainID,
		Config:        &cfg,
		InstanceLabel: "PROPOSER",
	}

	tests := []struct {
		name    string
		mutate  func(*stellar.InitializeMCMSInput)
		wantErr string
	}{
		{
			name: "empty contract ID",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.ContractID = ""
			},
			wantErr: "contract ID is empty",
		},
		{
			name: "empty owner",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.Owner = ""
			},
			wantErr: "owner is empty",
		},
		{
			name: "empty chain ID",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.ChainID = ""
			},
			wantErr: "chain ID is empty",
		},
		{
			name: "nil config",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.Config = nil
			},
			wantErr: "config is nil",
		},
		{
			name: "empty instance label",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.InstanceLabel = ""
			},
			wantErr: "instance label is empty",
		},
		{
			name: "instance label too long",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.InstanceLabel = strings.Repeat("a", 33)
			},
			wantErr: "exceeds 32 bytes",
		},
		{
			name: "invalid config",
			mutate: func(in *stellar.InitializeMCMSInput) {
				invalidConfig := types.Config{}
				in.Config = &invalidConfig
			},
			wantErr: "validate stellar MCMS config",
		},
		{
			name: "unknown chain ID",
			mutate: func(in *stellar.InitializeMCMSInput) {
				in.ChainID = "unknown-chain-id"
			},
			wantErr: "get stellar network passphrase from chain ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dependency := stellarmocks.NewContractDeployer(t)
			deployer := stellar.NewDeployer(dependency)

			input := valid
			tt.mutate(&input)

			_, err := deployer.InitializeMCMS(t.Context(), input)

			require.ErrorContains(t, err, tt.wantErr)
			dependency.AssertNotCalled(t, "InvokeContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func testMCMSConfig(t *testing.T) types.Config {
	t.Helper()

	cfg, err := types.NewConfig(1, []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")}, nil)
	require.NoError(t, err)

	return cfg
}

func testContractID(t *testing.T, seed byte) string {
	t.Helper()

	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = seed + byte(i)
	}

	id, err := strkey.Encode(strkey.VersionByteContract, raw)
	require.NoError(t, err)

	return id
}

func testAccountAddress(t *testing.T) string {
	t.Helper()

	kp, err := keypair.Random()
	require.NoError(t, err)

	return kp.Address()
}
