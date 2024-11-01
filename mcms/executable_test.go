package mcms

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/smartcontractkit/mcms/sdk"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	proposal_core "github.com/smartcontractkit/mcms/internal/core/proposal"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	evm_config "github.com/smartcontractkit/mcms/sdk/evm/config"
	evm_mcms "github.com/smartcontractkit/mcms/sdk/evm/proposal/mcms"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: This should go to the EVM SDK
func setupSimulatedBackendWithMCMS(numSigners uint64) ([]*ecdsa.PrivateKey, []*bind.TransactOpts, *backends.SimulatedBackend, *bindings.ManyChainMultiSig, error) {
	// Generate a private key
	keys := make([]*ecdsa.PrivateKey, numSigners)
	auths := make([]*bind.TransactOpts, numSigners)
	for i := range numSigners {
		key, _ := crypto.GenerateKey()
		auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
		if err != nil {
			return nil, nil, nil, nil, err
		}
		auth.GasLimit = uint64(8000000)
		keys[i] = key
		auths[i] = auth
	}

	// Setup a simulated backend
	// TODO: remove deprecated call
	//nolint:staticcheck
	genesisAlloc := map[common.Address]core.GenesisAccount{}
	for _, auth := range auths {
		// TODO: remove deprecated call
		//nolint:staticcheck
		genesisAlloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(1e18)}
	}
	blockGasLimit := uint64(8000000)

	// TODO: remove deprecated call
	//nolint:staticcheck
	sim := backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)

	// Deploy a ManyChainMultiSig contract with any of the signers
	_, tx, mcmsObj, err := bindings.DeployManyChainMultiSig(auths[0], sim)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the contract to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Check the receipt status
	if receipt.Status != gethTypes.ReceiptStatusSuccessful {
		return nil, nil, nil, nil, errors.New("contract deployment failed")
	}

	// Set a valid config
	signers := make([]common.Address, numSigners)
	for i, auth := range auths {
		signers[i] = auth.From
	}

	// Set the quorum
	quorum, err := safecast.Uint64ToUint8(numSigners)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	cfg := &types.Config{
		Quorum:       quorum,
		Signers:      signers,
		GroupSigners: []types.Config{},
	}
	configurator := evm_config.EVMConfigurator{}
	evmConfig, err := configurator.SetConfigInputs(*cfg)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	signerAddresses := make([]common.Address, len(evmConfig.Signers))
	signerGroups := make([]uint8, len(evmConfig.Signers))
	for i, signer := range evmConfig.Signers {
		signerAddresses[i] = signer.Addr
		signerGroups[i] = signer.Group
	}
	tx, err = mcmsObj.SetConfig(auths[0], signerAddresses, signerGroups, evmConfig.GroupQuorums, evmConfig.GroupParents, false)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Mine a block
	sim.Commit()

	// Wait for the transaction to be mined
	_, err = bind.WaitMined(auths[0].Context, sim, tx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return keys, auths, sim, mcmsObj, nil
}

func TestExecutor_ExecuteE2E_SingleChainSingleSignerSingleTX_Success(t *testing.T) {
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
		Signatures:           []types.Signature{},
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

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm_mcms.NewEVMExecutor(encoders[TestChain1].(*evm_mcms.EVMEncoder), sim, auths[0]),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	assert.NotEqual(t, "", txHash)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	txHash, err = executable.Execute(0)
	require.NoError(t, err)
	assert.NotEqual(t, "", txHash)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)

	// // Check the state of the MCMS contract
	newOpCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	assert.NotNil(t, newOpCount)
	assert.Equal(t, uint64(1), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := timelock.GetRoleMemberCount(&bind.CallOpts{}, role)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1), proposerCount)
	proposer, err := timelock.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	require.NoError(t, err)
	assert.Equal(t, mcmsObj.Address().Hex(), proposer.Hex())
}

func TestExecutor_ExecuteE2E_SingleChainMultipleSignerSingleTX_Success(t *testing.T) {
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
		Signatures:           []types.Signature{},
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

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm_mcms.NewEVMExecutor(encoders[TestChain1].(*evm_mcms.EVMEncoder), sim, auths[0]),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	assert.NotEqual(t, "", txHash)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	txHash, err = executable.Execute(0)
	require.NoError(t, err)
	assert.NotEqual(t, "", txHash)
	sim.Commit()

	// Wait for the transaction to be mined
	receipt, err := bind.WaitMined(auths[0].Context, sim, tx)
	require.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)

	// Check the state of the MCMS contract
	newOpCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	assert.NotNil(t, newOpCount)
	assert.Equal(t, uint64(1), newOpCount.Uint64())

	// Check the state of the timelock contract
	proposerCount, err := timelock.GetRoleMemberCount(&bind.CallOpts{}, role)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1), proposerCount)
	proposer, err := timelock.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
	require.NoError(t, err)
	assert.Equal(t, mcmsObj.Address().Hex(), proposer.Hex())
}

func TestExecutor_ExecuteE2E_SingleChainSingleSignerMultipleTX_Success(t *testing.T) {
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
		Signatures:           []types.Signature{},
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

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm_mcms.NewEVMExecutor(encoders[TestChain1].(*evm_mcms.EVMEncoder), sim, auths[0]),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	assert.NotEqual(t, "", txHash)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	for i := range 4 {
		// Execute the proposal
		txHash, err = executable.Execute(i)
		require.NoError(t, err)
		assert.NotEqual(t, "", txHash)
		sim.Commit()

		// Wait for the transaction to be mined
		receipt, merr := bind.WaitMined(auths[0].Context, sim, tx)
		require.NoError(t, merr)
		assert.NotNil(t, receipt)
		assert.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
	}

	// Check the state of the MCMS contract
	newOpCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	assert.NotNil(t, newOpCount)
	// assert.Equal(t, uint64(4), newOpCount.Uint64())

	// Check the state of the timelock contract
	for _, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		roleCount, err := timelock.GetRoleMemberCount(&bind.CallOpts{}, role)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(1), roleCount)
		roleMember, err := timelock.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
		require.NoError(t, err)
		assert.Equal(t, mcmsObj.Address().Hex(), roleMember.Hex())
	}
}

func TestExecutor_ExecuteE2E_SingleChainMultipleSignerMultipleTX_Success(t *testing.T) {
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
		Signatures:           []types.Signature{},
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

	// Construct encoders
	encoders, err := proposal.GetEncoders(true)
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		TestChain1: evm_mcms.NewEVMExecutor(encoders[TestChain1].(*evm_mcms.EVMEncoder), sim, auths[0]),
	}

	// Construct executable
	executable, err := proposal.Executable(true, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(TestChain1)
	require.NoError(t, err)
	assert.NotEqual(t, "", txHash)
	sim.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmsObj.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	assert.Equal(t, root.Root, [32]byte(signable.GetTree().Root.Bytes()))
	assert.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	for i := range 4 {
		// Execute the proposal
		txHash, err = executable.Execute(i)
		require.NoError(t, err)
		assert.NotEqual(t, "", txHash)
		sim.Commit()

		// Wait for the transaction to be mined
		receipt, merr := bind.WaitMined(auths[0].Context, sim, tx)
		require.NoError(t, merr)
		assert.NotNil(t, receipt)
		assert.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
	}

	// Check the state of the MCMS contract
	newOpCount, err := mcmsObj.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	assert.NotNil(t, newOpCount)
	assert.Equal(t, uint64(4), newOpCount.Uint64())

	// Check the state of the timelock contract
	for _, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		roleCount, err := timelock.GetRoleMemberCount(&bind.CallOpts{}, role)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(1), roleCount)
		roleMember, err := timelock.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(0))
		require.NoError(t, err)
		assert.Equal(t, mcmsObj.Address().Hex(), roleMember.Hex())
	}
}
