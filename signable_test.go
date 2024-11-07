package mcms

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcms_core "github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	sdkmocks "github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
)

type inspectorMocks struct {
	inspector1 *sdkmocks.Inspector
	inspector2 *sdkmocks.Inspector
}

func Test_NewSignable(t *testing.T) {
	t.Parallel()

	var (
		inspector = mocks.NewInspector(t) // We only need this to fulfill the interface argument requirements
	)

	tests := []struct {
		name           string
		giveProposal   *Proposal
		giveInspectors map[types.ChainSelector]sdk.Inspector
		wantErr        string
	}{
		{
			name: "failure: could not get encoders from proposal (invalid chain selector)",
			giveProposal: &Proposal{
				BaseProposal: BaseProposal{
					OverridePreviousRoot: false,
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						types.ChainSelector(1): {},
					},
				},
			},
			giveInspectors: map[types.ChainSelector]sdk.Inspector{
				types.ChainSelector(1): inspector,
			},
			wantErr: "unable to create encoder: invalid chain ID: 1",
		},
		{
			name: "failure: could not generate tree from proposal (invalid additional values)",
			giveProposal: &Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {StartingOpCount: 5},
					},
				},
				Transactions: []types.ChainOperation{
					{
						ChainSelector: TestChain1,
						Operation: types.Operation{
							AdditionalFields: json.RawMessage([]byte(``)),
						},
					},
				},
			},
			giveInspectors: map[types.ChainSelector]sdk.Inspector{
				types.ChainSelector(1): inspector,
			},
			wantErr: "merkle tree generation error: unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewSignable(tt.giveProposal, tt.giveInspectors)

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

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
	proposal := Proposal{
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
	proposal.UseSimulatedBackend(true)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
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
	proposal := Proposal{
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
	proposal.UseSimulatedBackend(true)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
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
	proposal := Proposal{
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
	proposal.UseSimulatedBackend(true)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
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
	proposal := Proposal{
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
	proposal.UseSimulatedBackend(true)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
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
	proposal := Proposal{
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
	proposal.UseSimulatedBackend(true)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
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
	proposal := Proposal{
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
	proposal.UseSimulatedBackend(true)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{TestChain1: evm.NewEVMInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
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

func Test_Signable_AddSignature(t *testing.T) {
	t.Parallel()

	proposal := Proposal{}
	signable := &Signable{proposal: &proposal}

	require.Empty(t, proposal.Signatures)
	signable.AddSignature(types.Signature{})
	require.Len(t, proposal.Signatures, 1)
}

func Test_Signable_GetConfigs(t *testing.T) {
	t.Parallel()

	var (
		config1 = &types.Config{}
		config2 = &types.Config{}
	)

	tests := []struct {
		name           string
		give           Proposal
		giveInspectors func(*inspectorMocks) map[types.ChainSelector]sdk.Inspector
		want           map[types.ChainSelector]*types.Config
		wantErr        string
	}{
		{
			name: "success",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
						TestChain2: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(config1, nil)
				m.inspector2.EXPECT().GetConfig("0x02").Return(config2, nil)

				return map[types.ChainSelector]sdk.Inspector{
					TestChain1: m.inspector1,
					TestChain2: m.inspector2,
				}
			},
			want: map[types.ChainSelector]*types.Config{
				TestChain1: config1,
				TestChain2: config2,
			},
		},
		{
			name: "failure: no inspectors",
			give: Proposal{},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				return nil
			},
			wantErr: ErrInspectorsNotProvided.Error(),
		},
		{
			name: "failure: inspector not found",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				return map[types.ChainSelector]sdk.Inspector{}
			},
			wantErr: "inspector not found for chain 3379446385462418246",
		},
		{
			name: "failure: on chain get config failure",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(nil, assert.AnError)

				return map[types.ChainSelector]sdk.Inspector{
					TestChain1: m.inspector1,
				}
			},
			wantErr: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector1 := sdkmocks.NewInspector(t)
			inspector2 := sdkmocks.NewInspector(t)

			giveInspectors := tt.giveInspectors(&inspectorMocks{
				inspector1: inspector1,
				inspector2: inspector2,
			})

			signable := &Signable{
				proposal:   &tt.give,
				inspectors: giveInspectors,
			}

			configs, err := signable.GetConfigs()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, configs)
			}
		})
	}
}

func Test_Signable_ValidateConfigs(t *testing.T) {
	t.Parallel()

	var (
		signer1 = common.HexToAddress("0x1")
		signer2 = common.HexToAddress("0x2")

		config1 = &types.Config{
			Quorum:  1,
			Signers: []common.Address{signer1},
		}
		config2 = &types.Config{ // Same as config1
			Quorum:  1,
			Signers: []common.Address{signer1},
		}
		config3 = &types.Config{ // Different from config1
			Quorum:  2,
			Signers: []common.Address{signer1, signer2},
		}
	)

	tests := []struct {
		name           string
		give           Proposal
		giveInspectors func(*inspectorMocks) map[types.ChainSelector]sdk.Inspector
		wantErr        string
	}{
		{
			name: "success",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
						TestChain2: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(config1, nil)
				m.inspector2.EXPECT().GetConfig("0x02").Return(config2, nil)

				return map[types.ChainSelector]sdk.Inspector{
					TestChain1: m.inspector1,
					TestChain2: m.inspector2,
				}
			},
		},
		{
			name: "failure: could not get configs",
			give: Proposal{},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				return nil
			},
			wantErr: ErrInspectorsNotProvided.Error(),
		},
		{
			name: "failure: not equal",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
						TestChain2: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(config1, nil)
				m.inspector2.EXPECT().GetConfig("0x02").Return(config3, nil)

				return map[types.ChainSelector]sdk.Inspector{
					TestChain1: m.inspector1,
					TestChain2: m.inspector2,
				}
			},
			wantErr: "inconsistent configs for chains 16015286601757825753 and 3379446385462418246",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector1 := sdkmocks.NewInspector(t)
			inspector2 := sdkmocks.NewInspector(t)

			giveInspectors := tt.giveInspectors(&inspectorMocks{
				inspector1: inspector1,
				inspector2: inspector2,
			})

			signable := &Signable{
				proposal:   &tt.give,
				inspectors: giveInspectors,
			}

			err := signable.ValidateConfigs()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_Signable_getCurrentOpCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		give           Proposal
		giveInspectors func(*inspectorMocks) map[types.ChainSelector]sdk.Inspector
		want           map[types.ChainSelector]uint64
		wantErr        string
	}{
		{
			name: "success",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
						TestChain2: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetOpCount("0x01").Return(100, nil)
				m.inspector2.EXPECT().GetOpCount("0x02").Return(200, nil)

				return map[types.ChainSelector]sdk.Inspector{
					TestChain1: m.inspector1,
					TestChain2: m.inspector2,
				}
			},
			want: map[types.ChainSelector]uint64{
				TestChain1: 100,
				TestChain2: 200,
			},
		},
		{
			name: "failure: could not get configs",
			give: Proposal{},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				return nil
			},
			wantErr: ErrInspectorsNotProvided.Error(),
		},
		{
			name: "failure: inspector not found",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				return map[types.ChainSelector]sdk.Inspector{}
			},
			wantErr: "inspector not found for chain 3379446385462418246",
		},
		{
			name: "failure: on chain GetOpCount failure",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						TestChain1: {MCMAddress: "0x01"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetOpCount("0x01").Return(0, assert.AnError)

				return map[types.ChainSelector]sdk.Inspector{
					TestChain1: m.inspector1,
				}
			},
			wantErr: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector1 := sdkmocks.NewInspector(t)
			inspector2 := sdkmocks.NewInspector(t)

			giveInspectors := tt.giveInspectors(&inspectorMocks{
				inspector1: inspector1,
				inspector2: inspector2,
			})

			signable := &Signable{
				proposal:   &tt.give,
				inspectors: giveInspectors,
			}

			got, err := signable.getCurrentOpCounts()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
