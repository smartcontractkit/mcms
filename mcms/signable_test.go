package mcms

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	mcms_core "github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: Should go to EVM SDK
func TestSignable_SingleChainSingleSignerSingleTX_Success(t *testing.T) {
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
		BaseProposal: BaseProposal{
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

	err = Sign(signable, NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)
}

func TestSignable_SingleChainMultipleSignerSingleTX_Success(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 3)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelockC contract for testing
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
		BaseProposal: BaseProposal{
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
	for _, s := range sim.Signers {
		err = Sign(signable, NewPrivateKeySigner(s.PrivateKey))
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)
}

func TestSignable_SingleChainSingleSignerMultipleTX_Success(t *testing.T) {
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
		BaseProposal: BaseProposal{
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
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	err = Sign(signable, NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)
}

func TestSignable_SingleChainMultipleSignerMultipleTX_Success(t *testing.T) {
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
		BaseProposal: BaseProposal{
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
		err = Sign(signable, NewPrivateKeySigner(sim.Signers[i].PrivateKey))
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.NoError(t, err)
	require.True(t, quorumMet)
}

func TestSignable_SingleChainMultipleSignerMultipleTX_FailureMissingQuorum(t *testing.T) {
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
		BaseProposal: BaseProposal{
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
	for _, s := range sim.Signers[:2] { // Only sign with 2 out of 3 signers
		err = Sign(signable, NewPrivateKeySigner(s.PrivateKey))
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.Error(t, err)
	require.IsType(t, &QuorumNotReachedError{}, err)
	require.False(t, quorumMet)
}

func TestSignable_SingleChainMultipleSignerMultipleTX_FailureInvalidSigner(t *testing.T) {
	t.Parallel()

	sim := evmsim.NewSimulatedChain(t, 3)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelockC contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address())

	// Generate a new key for an invalid signer
	invalidKey, err := crypto.GenerateKey()
	require.NoError(t, err)

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
		BaseProposal: BaseProposal{
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
		},
		Transactions: operations,
	}

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := proposal.Signable(true, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash with all signers
	for _, s := range sim.Signers {
		err = Sign(signable, NewPrivateKeySigner(s.PrivateKey))
		require.NoError(t, err)
	}

	// Sign with the invalid signer
	err = Sign(signable, NewPrivateKeySigner(invalidKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.Error(t, err)
	require.IsType(t, &mcms_core.InvalidSignatureError{}, err)
	require.False(t, quorumMet)
}
