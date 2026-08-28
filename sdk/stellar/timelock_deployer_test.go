package stellar_test

import (
	"errors"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

	_, err := deployer.InitializeTimelock(
		t.Context(),
		input,
	)
	require.NoError(t, err)
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

	_, err := deployer.InitializeTimelock(
		t.Context(),
		input,
	)
	require.NoError(t, err)
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
		stellar.InitializeTimelockInput{
			ContractID: testContractID(t, 10),
			Proposers: []string{
				testContractID(t, 20),
			},
			Cancellers: []string{
				testContractID(t, 20),
			},
			Bypassers: []string{
				testContractID(t, 30),
			},
		},
	)

	require.ErrorIs(t, err, expectedErr)
	require.ErrorContains(
		t,
		err,
		"initialize Stellar timelock",
	)
}

func TestTimelockDeployer_InitializeTimelock_EmptyContractID(
	t *testing.T,
) {
	t.Parallel()

	dependency := stellarmocks.NewContractDeployer(t)
	deployer := stellar.NewTimelockDeployer(dependency)

	_, err := deployer.InitializeTimelock(
		t.Context(),
		stellar.InitializeTimelockInput{},
	)

	require.ErrorContains(t, err, "contract ID is empty")

	dependency.AssertNotCalled(
		t,
		"InvokeContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}
