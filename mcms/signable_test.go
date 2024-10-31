package mcms

import (
	"github.com/smartcontractkit/mcms/sdk"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcms_core "github.com/smartcontractkit/mcms/internal/core"
	proposal_core "github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_mcms "github.com/smartcontractkit/mcms/sdk/evm/proposal/mcms"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: Should go to EVM SDK
func TestSignable_SingleChainSingleSignerSingleTX_Success(t *testing.T) {
	t.Parallel()

	keys, auths, sim, mcmsObj, err := setupSimulatedBackendWithMCMS(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)

	// Deploy a timelock contract for testing
	addr, tx, timelock, err := bindings.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		mcmsObj.Address(),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	require.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, tx)
	assert.NotNil(t, timelock)
	sim.Commit()

	// Construct example transaction
	role, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, mcmsObj.Address())
	require.NoError(t, err)

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []mcms.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address().Hex(),
			},
		},
		Transactions: []types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: evm_mcms.NewEVMOperation(
					timelock.Address(),
					grantRoleData,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm_mcms.NewEVMInspector(sim)}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	assert.NotNil(t, signable)

	err = proposal_core.SignPlainKey(keys[0], &proposal, true, inspectors)
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	assert.True(t, quorumMet)
	require.NoError(t, err)
}

func TestSignable_SingleChainMultipleSignerSingleTX_Success(t *testing.T) {
	t.Parallel()

	keys, auths, sim, mcmsObj, err := setupSimulatedBackendWithMCMS(3)
	require.NoError(t, err)
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	for i := range 3 {
		assert.NotNil(t, keys[i])
		assert.NotNil(t, auths[i])
	}

	// Deploy a timelock contract for testing
	addr, tx, timelock, err := bindings.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		mcmsObj.Address(),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	require.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, tx)
	assert.NotNil(t, timelock)
	sim.Commit()

	// Construct example transaction
	role, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, mcmsObj.Address())
	require.NoError(t, err)

	// Construct a proposal
	proposal := MCMSProposal{
		Version:              "1.0",
		Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
		ValidUntil:           2004259681,
		Signatures:           []mcms.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address().Hex(),
			},
		},
		Transactions: []types.ChainOperation{
			{
				ChainSelector: TestChain1,
				Operation: evm_mcms.NewEVMOperation(
					timelock.Address(),
					grantRoleData,
					big.NewInt(0),
					"Sample contract",
					[]string{"tag1", "tag2"},
				),
			},
		},
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm_mcms.NewEVMInspector(sim)}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	assert.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		err = proposal_core.SignPlainKey(keys[i], &proposal, true, inspectors)
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	assert.True(t, quorumMet)
	require.NoError(t, err)
}

func TestSignable_SingleChainSingleSignerMultipleTX_Success(t *testing.T) {
	t.Parallel()

	keys, auths, sim, mcmsObj, err := setupSimulatedBackendWithMCMS(1)
	require.NoError(t, err)
	assert.NotNil(t, keys[0])
	assert.NotNil(t, auths[0])
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)

	// Deploy a timelock contract for testing
	addr, tx, timelock, err := bindings.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		mcmsObj.Address(),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	require.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, tx)
	assert.NotNil(t, timelock)
	sim.Commit()

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	executorRole, err := timelock.EXECUTORROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]types.ChainOperation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmsObj.Address())
		require.NoError(t, perr)
		operations[i] = types.ChainOperation{
			ChainSelector: TestChain1,
			Operation: evm_mcms.NewEVMOperation(
				timelock.Address(),
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
		Signatures:           []mcms.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address().Hex(),
			},
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm_mcms.NewEVMInspector(sim)}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	assert.NotNil(t, signable)

	err = proposal_core.SignPlainKey(keys[0], &proposal, true, inspectors)
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	assert.True(t, quorumMet)
	require.NoError(t, err)
}

func TestSignable_SingleChainMultipleSignerMultipleTX_Success(t *testing.T) {
	t.Parallel()

	keys, auths, sim, mcmsObj, err := setupSimulatedBackendWithMCMS(3)
	require.NoError(t, err)
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	for i := range 3 {
		assert.NotNil(t, keys[i])
		assert.NotNil(t, auths[i])
	}

	// Deploy a timelock contract for testing
	addr, tx, timelock, err := bindings.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		mcmsObj.Address(),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	require.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, tx)
	assert.NotNil(t, timelock)
	sim.Commit()

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	executorRole, err := timelock.EXECUTORROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]types.ChainOperation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmsObj.Address())
		require.NoError(t, perr)
		operations[i] = types.ChainOperation{
			ChainSelector: TestChain1,
			Operation: evm_mcms.NewEVMOperation(
				timelock.Address(),
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
		Signatures:           []mcms.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address().Hex(),
			},
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm_mcms.NewEVMInspector(sim)}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	assert.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		err = proposal_core.SignPlainKey(keys[i], &proposal, true, inspectors)
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	assert.True(t, quorumMet)
	require.NoError(t, err)
}

func TestSignable_SingleChainMultipleSignerMultipleTX_FailureMissingQuorum(t *testing.T) {
	t.Parallel()

	keys, auths, sim, mcmsObj, err := setupSimulatedBackendWithMCMS(3)
	require.NoError(t, err)
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	for i := range 3 {
		assert.NotNil(t, keys[i])
		assert.NotNil(t, auths[i])
	}

	// Deploy a timelock contract for testing
	addr, tx, timelock, err := bindings.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		mcmsObj.Address(),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	require.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, tx)
	assert.NotNil(t, timelock)
	sim.Commit()

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	executorRole, err := timelock.EXECUTORROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]types.ChainOperation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmsObj.Address())
		require.NoError(t, perr)
		operations[i] = types.ChainOperation{
			ChainSelector: TestChain1,
			Operation: evm_mcms.NewEVMOperation(
				timelock.Address(),
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
		Signatures:           []mcms.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address().Hex(),
			},
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm_mcms.NewEVMInspector(sim)}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	assert.NotNil(t, signable)

	// Sign the hash
	for i := range 2 {
		err = proposal_core.SignPlainKey(keys[i], &proposal, true, inspectors)
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	assert.False(t, quorumMet)
	require.Error(t, err)
	// assert error is of type QuorumNotMetError
	assert.IsType(t, &mcms_core.QuorumNotMetError{}, err)
}

func TestSignable_SingleChainMultipleSignerMultipleTX_FailureInvalidSigner(t *testing.T) {
	t.Parallel()

	keys, auths, sim, mcmsObj, err := setupSimulatedBackendWithMCMS(3)
	require.NoError(t, err)
	assert.NotNil(t, sim)
	assert.NotNil(t, mcmsObj)
	for i := range 3 {
		assert.NotNil(t, keys[i])
		assert.NotNil(t, auths[i])
	}

	// Generate a new key
	newKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	keys[2] = newKey

	// Deploy a timelock contract for testing
	addr, tx, timelock, err := bindings.DeployRBACTimelock(
		auths[0],
		sim,
		big.NewInt(0),
		mcmsObj.Address(),
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
		[]common.Address{},
	)
	require.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, tx)
	assert.NotNil(t, timelock)
	sim.Commit()

	// Construct example transactions
	proposerRole, err := timelock.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	bypasserRole, err := timelock.BYPASSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	cancellerRole, err := timelock.CANCELLERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	executorRole, err := timelock.EXECUTORROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	operations := make([]types.ChainOperation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmsObj.Address())
		require.NoError(t, perr)
		operations[i] = types.ChainOperation{
			ChainSelector: TestChain1,
			Operation: evm_mcms.NewEVMOperation(
				timelock.Address(),
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
		Signatures:           []mcms.Signature{},
		OverridePreviousRoot: false,
		ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
			TestChain1: {
				StartingOpCount: 0,
				MCMAddress:      mcmsObj.Address().Hex(),
			},
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm_mcms.NewEVMInspector(sim)}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	assert.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		err = proposal_core.SignPlainKey(keys[i], &proposal, true, inspectors)
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	assert.False(t, quorumMet)
	require.Error(t, err)
	// assert error is of type QuorumNotMetError
	assert.IsType(t, &mcms_core.InvalidSignatureError{}, err)
}
