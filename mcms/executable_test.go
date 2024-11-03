package mcms

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	proposal_core "github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: This should go to the EVM SDK
func TestExecutor_ExecuteE2E_SingleChainSingleSignerSingleTX_Success(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 1)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelock contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address())

	// Construct example transaction
	role, err := timelockC.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, mcmC.Address())
	require.NoError(t, err)

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []types.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmC.Address().Hex(),
			},
		},
		Transactions: []types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: evm.NewEVMOperation(
					timelockC.Address(),
					grantRoleData,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	err = proposal_core.SignPlainKey(sim.Signers[0].PrivateKey, &proposal, true, inspectors)
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm.NewEVMExecutor(encoders[TestChain1].(*evm.EVMEncoder), sim.Backend.Client(), sim.Signers[0].NewTransactOpts(t)),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	txHash, err = executable.Execute(0)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)
	sim.Backend.Commit()

	// Check the state of the MCMS contract
	newOpCount, err := mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(1), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := timelockC.GetRoleMemberCount(&bind.CallOpts{}, role)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), proposerCount)
	proposer, err := timelockC.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, mcmC.Address().Hex(), proposer.Hex())
}

func TestExecutor_ExecuteE2E_SingleChainMultipleSignerSingleTX_Success(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 3)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelock contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address())

	// Construct example transaction
	role, err := timelockC.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, mcmC.Address())
	require.NoError(t, err)

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []types.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmC.Address().Hex(),
			},
		},
		Transactions: []types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: evm.NewEVMOperation(
					timelockC.Address(),
					grantRoleData,
					big.NewInt(0),
					"Sample contract",
					[]string{"tag1", "tag2"},
				),
			},
		},
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		err = proposal_core.SignPlainKey(sim.Signers[i].PrivateKey, &proposal, true, inspectors)
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm.NewEVMExecutor(encoders[TestChain1].(*evm.EVMEncoder), sim.Backend.Client(), sim.Signers[0].NewTransactOpts(t)),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	txHash, err = executable.Execute(0)
	require.NoError(t, err)
	require.NotEqual(t, "", txHash)
	sim.Backend.Commit()

	// Check the state of the MCMS contract
	newOpCount, err := mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(1), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := timelockC.GetRoleMemberCount(&bind.CallOpts{}, role)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), proposerCount)
	proposer, err := timelockC.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, mcmC.Address().Hex(), proposer.Hex())
}

func TestExecutor_ExecuteE2E_SingleChainSingleSignerMultipleTX_Success(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 1)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelockC contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address())

	// Construct example transactions
	proposerRole, err := timelockC.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelockC.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelockC.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	executorRole, err := timelockC.EXECUTORROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]types.ChainOperation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)

		operations[i] = types.ChainOperation{
			ChainSelector: TestChain1,
			Operation: evm.NewEVMOperation(
				timelockC.Address(),
				data,
				big.NewInt(0),
				"Sample contract",
				[]string{"tag1", "tag2"},
			),
		}
	}

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer','Canceller','Executor', and 'Bypasser' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []types.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmC.Address().Hex(),
			},
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	err = proposal_core.SignPlainKey(sim.Signers[0].PrivateKey, &proposal, true, inspectors)
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm.NewEVMExecutor(encoders[TestChain1].(*evm.EVMEncoder), sim.Backend.Client(), sim.Signers[0].NewTransactOpts(t)),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	for i := range 4 {
		// Execute the proposal
		txHash, err = executable.Execute(i)
		require.NoError(t, err)
		require.NotEqual(t, "", txHash)

		sim.Backend.Commit()
	}

	// Check the state of the MCMS contract
	newOpCount, err := mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(4), newOpCount.Uint64())

	// Check the state of the timelock contract
	for _, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		roleCount, err := timelockC.GetRoleMemberCount(&bind.CallOpts{}, role)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1), roleCount)

		roleMember, err := timelockC.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
		require.NoError(t, err)
		require.Equal(t, mcmC.Address().Hex(), roleMember.Hex())
	}
}

func TestExecutor_ExecuteE2E_SingleChainMultipleSignerMultipleTX_Success(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 3)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelockC contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address())

	// Construct example transactions
	proposerRole, err := timelockC.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelockC.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelockC.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	executorRole, err := timelockC.EXECUTORROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]types.ChainOperation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)
		operations[i] = types.ChainOperation{
			ChainSelector: TestChain1,
			Operation: evm.NewEVMOperation(
				timelockC.Address(),
				data,
				big.NewInt(0),
				"Sample contract",
				[]string{"tag1", "tag2"},
			),
		}
	}

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer','Canceller','Executor', and 'Bypasser' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []types.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmC.Address().Hex(),
			},
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		err = proposal_core.SignPlainKey(sim.Signers[i].PrivateKey, &proposal, true, inspectors)
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm.NewEVMExecutor(encoders[TestChain1].(*evm.EVMEncoder), sim.Backend.Client(), sim.Signers[0].NewTransactOpts(t)),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)

	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	for i := range 4 {
		// Execute the proposal
		txHash, err = executable.Execute(i)
		require.NoError(t, err)
		require.NotEqual(t, "", txHash)

		sim.Backend.Commit()
	}

	// Check the state of the MCMS contract
	newOpCount, err := mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(4), newOpCount.Uint64())

	// Check the state of the timelock contract
	for _, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		roleCount, err := timelockC.GetRoleMemberCount(&bind.CallOpts{}, role)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1), roleCount)

		roleMember, err := timelockC.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
		require.NoError(t, err)
		require.Equal(t, mcmC.Address().Hex(), roleMember.Hex())
	}
}
