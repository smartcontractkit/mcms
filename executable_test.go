package mcms

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func Test_NewExecutable(t *testing.T) {
	t.Parallel()

	var (
		executor = mocks.NewExecutor(t) // We only need this to fulfill the interface argument requirements
	)

	tests := []struct {
		name          string
		giveProposal  *Proposal
		giveExecutors map[types.ChainSelector]sdk.Executor
		wantErr       string
	}{
		{
			name: "failure: could not get encoders from proposal (invalid chain selector)",
			giveProposal: &Proposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					Kind:                 types.KindProposal,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(1): {},
					},
				},
			},
			giveExecutors: map[types.ChainSelector]sdk.Executor{
				types.ChainSelector(1): executor,
			},
			wantErr: "unable to create encoder: chain family not found for selector 1",
		},
		{
			name: "failure: could not generate tx nonces from proposal (tx does not have matching chain metadata)",
			giveProposal: &Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain1Selector: {StartingOpCount: 5},
					},
				},
				Operations: []types.Operation{
					{ChainSelector: chaintest.Chain2Selector},
				},
			},
			giveExecutors: map[types.ChainSelector]sdk.Executor{
				types.ChainSelector(1): executor,
			},
			wantErr: "missing metadata for chain 16015286601757825753",
		},
		{
			name: "failure: could not generate tree from proposal (invalid additional values)",
			giveProposal: &Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain1Selector: {StartingOpCount: 5},
					},
				},
				Operations: []types.Operation{
					{
						ChainSelector: chaintest.Chain1Selector,
						Transaction: types.Transaction{
							AdditionalFields: json.RawMessage([]byte(``)),
						},
					},
				},
			},
			giveExecutors: map[types.ChainSelector]sdk.Executor{
				types.ChainSelector(1): executor,
			},
			wantErr: "merkle tree generation error: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewExecutable(tt.giveProposal, tt.giveExecutors)

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

// TODO: This should go to the EVM SDK
func TestExecutor_ExecuteE2E_SingleChainSingleSignerSingleTX_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sim := evmsim.NewSimulatedChain(t, 1)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelock contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address(), []common.Address{}, []common.Address{}, []common.Address{}, []common.Address{})

	// Construct example transaction
	role, err := timelockC.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, mcmC.Address())
	require.NoError(t, err)

	// Construct a proposal
	proposal := Proposal{
		BaseProposal: BaseProposal{
			Version:              "v1",
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			Kind:                 types.KindProposal,
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain1Selector: {
					StartingOpCount: 0,
					MCMAddress:      mcmC.Address().Hex(),
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: evm.NewTransaction(
					timelockC.Address(),
					grantRoleData,
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}
	proposal.UseSimulatedBackend(true)

	tree, err := proposal.MerkleTree()
	require.NoError(t, err)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{
		chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client()),
	}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		chaintest.Chain1Selector: evm.NewExecutor(
			encoders[chaintest.Chain1Selector].(*evm.Encoder),
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Construct executable
	executable, err := NewExecutable(&proposal, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotEmpty(t, tx.Hash)
	require.NotNil(t, tx.RawData)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(tree.Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executable.Execute(ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotEmpty(t, tx.Hash)
	require.NotNil(t, tx.RawData)
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

	ctx := context.Background()
	sim := evmsim.NewSimulatedChain(t, 3)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelock contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address(), []common.Address{}, []common.Address{}, []common.Address{}, []common.Address{})

	// Construct example transaction
	role, err := timelockC.PROPOSERROLE(&bind.CallOpts{})
	require.NoError(t, err)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)
	grantRoleData, err := timelockAbi.Pack("grantRole", role, mcmC.Address())
	require.NoError(t, err)

	// Construct a proposal
	proposal := Proposal{
		BaseProposal: BaseProposal{
			Version:              "v1",
			Kind:                 types.KindProposal,
			Description:          "Grants RBACTimelock 'Proposer' Role to MCMS Contract",
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain1Selector: {
					StartingOpCount: 0,
					MCMAddress:      mcmC.Address().Hex(),
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: evm.NewTransaction(
					timelockC.Address(),
					grantRoleData,
					big.NewInt(0),
					"Sample contract",
					[]string{"tag1", "tag2"},
				),
			},
		},
	}
	proposal.UseSimulatedBackend(true)

	tree, err := proposal.MerkleTree()
	require.NoError(t, err)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{
		chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client()),
	}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[i].PrivateKey))
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		chaintest.Chain1Selector: evm.NewExecutor(
			encoders[chaintest.Chain1Selector].(*evm.Encoder),
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Construct executable
	executable, err := NewExecutable(&proposal, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotEmpty(t, tx.Hash)
	require.NotNil(t, tx.RawData)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(tree.Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	tx, err = executable.Execute(ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, tx.RawData)
	require.NotEmpty(t, tx.Hash)
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

	ctx := context.Background()
	sim := evmsim.NewSimulatedChain(t, 1)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelockC contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address(), []common.Address{}, []common.Address{}, []common.Address{}, []common.Address{})

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

	operations := make([]types.Operation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)

		operations[i] = types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewTransaction(
				timelockC.Address(),
				data,
				big.NewInt(0),
				"Sample contract",
				[]string{"tag1", "tag2"},
			),
		}
	}

	// Construct a proposal
	proposal := Proposal{
		BaseProposal: BaseProposal{
			Version:              "v1",
			Kind:                 types.KindProposal,
			Description:          "Grants RBACTimelock 'Proposer','Canceller','Executor', and 'Bypasser' Role to MCMS Contract",
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain1Selector: {
					StartingOpCount: 0,
					MCMAddress:      mcmC.Address().Hex(),
				},
			},
		},
		Operations: operations,
	}
	proposal.UseSimulatedBackend(true)

	tree, err := proposal.MerkleTree()
	require.NoError(t, err)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{
		chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client()),
	}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		chaintest.Chain1Selector: evm.NewExecutor(
			encoders[chaintest.Chain1Selector].(*evm.Encoder),
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Construct executable
	executable, err := NewExecutable(&proposal, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, tx.RawData)
	require.NotEmpty(t, tx.Hash)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(tree.Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	for i := range 4 {
		// Execute the proposal
		tx, err = executable.Execute(ctx, i)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.NotNil(t, tx.RawData)
		require.NotEmpty(t, tx.Hash)

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

	ctx := context.Background()
	sim := evmsim.NewSimulatedChain(t, 3)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelockC contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], mcmC.Address(), []common.Address{}, []common.Address{}, []common.Address{}, []common.Address{})

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

	operations := make([]types.Operation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)
		operations[i] = types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewTransaction(
				timelockC.Address(),
				data,
				big.NewInt(0),
				"Sample contract",
				[]string{"tag1", "tag2"},
			),
		}
	}

	// Construct a proposal
	proposal := Proposal{
		BaseProposal: BaseProposal{
			Version:              "v1",
			Kind:                 types.KindProposal,
			Description:          "Grants RBACTimelock 'Proposer','Canceller','Executor', and 'Bypasser' Role to MCMS Contract",
			ValidUntil:           2004259681,
			Signatures:           []types.Signature{},
			OverridePreviousRoot: false,
			ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
				chaintest.Chain1Selector: {
					StartingOpCount: 0,
					MCMAddress:      mcmC.Address().Hex(),
				},
			},
		},
		Operations: operations,
	}
	proposal.UseSimulatedBackend(true)

	tree, err := proposal.MerkleTree()
	require.NoError(t, err)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{
		chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client()),
	}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash
	for i := range 3 {
		_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[i].PrivateKey))
		require.NoError(t, err)
	}

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := proposal.GetEncoders()
	require.NoError(t, err)

	// Construct executors
	executors := map[types.ChainSelector]sdk.Executor{
		chaintest.Chain1Selector: evm.NewExecutor(
			encoders[chaintest.Chain1Selector].(*evm.Encoder),
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Construct executable
	executable, err := NewExecutable(&proposal, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, tx.RawData)
	require.NotEmpty(t, tx.Hash)

	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(tree.Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	for i := range 4 {
		// Execute the proposal
		tx, err = executable.Execute(ctx, i)
		require.NoError(t, err)
		require.NotNil(t, tx)
		require.NotNil(t, tx.RawData)
		require.NotEmpty(t, tx.Hash)

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
