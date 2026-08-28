package stellar_test

import (
	"errors"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms/sdk/stellar"
	stellarmocks "github.com/smartcontractkit/mcms/sdk/stellar/mocks"
)

func TestTimelockDeployer_DeployTimelock(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	const wasmPath = "/tmp/timelock.wasm"

	salt := [32]byte{4, 3, 2, 1}
	contractID := testContractID(t, 10)

	dependency.EXPECT().
		DeployContract(
			mock.Anything,
			wasmPath,
			salt,
		).
		Return(contractID, nil).
		Once()

	deployer := stellar.NewTimelockDeployer(dependency)

	got, err := deployer.DeployTimelock(
		t.Context(),
		wasmPath,
		salt,
	)
	require.NoError(t, err)
	require.Equal(t, contractID, got)
}

func TestTimelockDeployer_DeployTimelock_DeployError(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	expectedErr := errors.New("deploy failed")

	dependency.EXPECT().
		DeployContract(
			mock.Anything,
			"/tmp/timelock.wasm",
			[32]byte{},
		).
		Return("", expectedErr).
		Once()

	deployer := stellar.NewTimelockDeployer(dependency)

	_, err := deployer.DeployTimelock(
		t.Context(),
		"/tmp/timelock.wasm",
		[32]byte{},
	)

	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(t, err, "deploy Stellar timelock")
}

func TestTimelockDeployer_DeployTimelock_EmptyWASMPath(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	deployer := stellar.NewTimelockDeployer(dependency)

	_, err := deployer.DeployTimelock(
		t.Context(),
		"",
		[32]byte{},
	)

	require.ErrorContains(t, err, "WASM path is empty")
	dependency.AssertNotCalled(
		t,
		"DeployContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestTimelockDeployer_DeployTimelock_EmptyContractID(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	dependency.EXPECT().
		DeployContract(
			mock.Anything,
			"/tmp/timelock.wasm",
			[32]byte{},
		).
		Return("", nil).
		Once()

	deployer := stellar.NewTimelockDeployer(dependency)

	_, err := deployer.DeployTimelock(
		t.Context(),
		"/tmp/timelock.wasm",
		[32]byte{},
	)

	require.ErrorContains(t, err, "empty contract ID")
}

func TestTimelockDeployer_DeployTimelock_NilDeployer(t *testing.T) {
	t.Parallel()

	deployer := stellar.NewTimelockDeployer(nil)

	_, err := deployer.DeployTimelock(
		t.Context(),
		"/tmp/timelock.wasm",
		[32]byte{},
	)

	require.ErrorContains(t, err, "deployer is nil")
}

func TestTimelockDeployer_InitializeTimelock(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	input := stellar.InitializeTimelockInput{
		ContractID: testContractID(t, 10),
		MinDelay:   123,
		Proposers: []string{
			testContractID(t, 20),
		},
		Cancellers: []string{
			testContractID(t, 20),
			testContractID(t, 30),
		},
		Bypassers: []string{
			testContractID(t, 40),
		},
	}

	expectedArgs := []xdr.ScVal{
		scval.Uint64ToScVal(input.MinDelay),
		scval.AddressSliceToScVal(input.Proposers),
		scval.AddressSliceToScVal(input.Cancellers),
		scval.AddressSliceToScVal(input.Bypassers),
	}

	dependency.EXPECT().
		InvokeContract(
			mock.Anything,
			input.ContractID,
			"initialize",
			expectedArgs,
		).
		Return((*xdr.ScVal)(nil), nil).
		Once()

	deployer := stellar.NewTimelockDeployer(dependency)

	result, err := deployer.InitializeTimelock(
		t.Context(),
		input,
	)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, result.ChainFamily)
}

func TestTimelockDeployer_InitializeTimelock_ZeroDelay(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)

	input := stellar.InitializeTimelockInput{
		ContractID: testContractID(t, 10),
		MinDelay:   0,
		Proposers: []string{
			testContractID(t, 20),
		},
		Cancellers: []string{
			testContractID(t, 20),
		},
		Bypassers: []string{
			testContractID(t, 30),
		},
	}

	dependency.EXPECT().
		InvokeContract(
			mock.Anything,
			input.ContractID,
			"initialize",
			[]xdr.ScVal{
				scval.Uint64ToScVal(0),
				scval.AddressSliceToScVal(input.Proposers),
				scval.AddressSliceToScVal(input.Cancellers),
				scval.AddressSliceToScVal(input.Bypassers),
			},
		).
		Return((*xdr.ScVal)(nil), nil).
		Once()

	deployer := stellar.NewTimelockDeployer(dependency)

	result, err := deployer.InitializeTimelock(
		t.Context(),
		input,
	)
	require.NoError(t, err)
	require.Equal(t, chainselectors.FamilyStellar, result.ChainFamily)
}

func TestTimelockDeployer_InitializeTimelock_InvokeError(t *testing.T) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	expectedErr := errors.New("invoke failed")

	dependency.EXPECT().
		InvokeContract(
			mock.Anything,
			mock.Anything,
			"initialize",
			mock.Anything,
		).
		Return((*xdr.ScVal)(nil), expectedErr).
		Once()

	deployer := stellar.NewTimelockDeployer(dependency)

	_, err := deployer.InitializeTimelock(
		t.Context(),
		validTimelockInitializeInput(t),
	)

	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(t, err, "initialize Stellar timelock")
}

func TestTimelockDeployer_InitializeTimelock_NilDeployer(t *testing.T) {
	t.Parallel()

	deployer := stellar.NewTimelockDeployer(nil)

	_, err := deployer.InitializeTimelock(
		t.Context(),
		stellar.InitializeTimelockInput{},
	)

	require.ErrorContains(t, err, "deployer is nil")
}

func TestTimelockDeployer_InitializeTimelock_InvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*testing.T, *stellar.InitializeTimelockInput)
		wantErr string
	}{
		{
			name: "empty contract ID",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.ContractID = ""
			},
			wantErr: "contract ID is empty",
		},
		{
			name: "empty proposers",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.Proposers = nil
			},
			wantErr: "proposers are empty",
		},
		{
			name: "empty cancellers",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.Cancellers = nil
			},
			wantErr: "cancellers are empty",
		},
		{
			name: "empty bypassers",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.Bypassers = nil
			},
			wantErr: "bypassers are empty",
		},
		{
			name: "empty proposer address",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.Proposers = []string{""}
			},
			wantErr: "proposers[0] is empty",
		},
		{
			name: "empty canceller address",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.Cancellers = []string{""}
			},
			wantErr: "cancellers[0] is empty",
		},
		{
			name: "empty bypasser address",
			mutate: func(_ *testing.T, in *stellar.InitializeTimelockInput) {
				in.Bypassers = []string{""}
			},
			wantErr: "bypassers[0] is empty",
		},
		{
			name: "duplicate proposer",
			mutate: func(t *testing.T, in *stellar.InitializeTimelockInput) {
				address := testContractID(t, 90)
				in.Proposers = []string{address, address}
			},
			wantErr: "proposers contains duplicate address",
		},
		{
			name: "duplicate canceller",
			mutate: func(t *testing.T, in *stellar.InitializeTimelockInput) {
				address := testContractID(t, 91)
				in.Cancellers = []string{address, address}
			},
			wantErr: "cancellers contains duplicate address",
		},
		{
			name: "duplicate bypasser",
			mutate: func(t *testing.T, in *stellar.InitializeTimelockInput) {
				address := testContractID(t, 92)
				in.Bypassers = []string{address, address}
			},
			wantErr: "bypassers contains duplicate address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dependency := stellarmocks.NewContractDeployer(t)
			deployer := stellar.NewTimelockDeployer(dependency)

			input := validTimelockInitializeInput(t)
			tt.mutate(t, &input)

			_, err := deployer.InitializeTimelock(
				t.Context(),
				input,
			)

			require.ErrorContains(t, err, tt.wantErr)
			dependency.AssertNotCalled(
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

func validTimelockInitializeInput(t *testing.T) stellar.InitializeTimelockInput {
	t.Helper()

	return stellar.InitializeTimelockInput{
		ContractID: testContractID(t, 10),
		MinDelay:   123,
		Proposers: []string{
			testContractID(t, 20),
		},
		Cancellers: []string{
			testContractID(t, 30),
		},
		Bypassers: []string{
			testContractID(t, 40),
		},
	}
}
