package mcms

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
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

type simulatorMocks struct {
	simulator1 *sdkmocks.Simulator
	simulator2 *sdkmocks.Simulator
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
			wantErr: "unable to create encoder: chain family not found for selector 1",
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
				Transaction: evm.NewOperation(
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
	inspectors := map[types.ChainSelector]sdk.Inspector{chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)

	_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
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
				Transaction: evm.NewOperation(
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
	inspectors := map[types.ChainSelector]sdk.Inspector{chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash
	for _, s := range sim.Signers {
		_, err = signable.SignAndAppend(NewPrivateKeySigner(s.PrivateKey))
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

	operations := make([]types.Operation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)
		operations[i] = types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewOperation(
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

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
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

	operations := make([]types.Operation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)
		operations[i] = types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewOperation(
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

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client())}

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

	operations := make([]types.Operation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)
		operations[i] = types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewOperation(
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

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash
	for _, s := range sim.Signers[:2] { // Only sign with 2 out of 3 signers
		_, err = signable.SignAndAppend(NewPrivateKeySigner(s.PrivateKey))
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

	operations := make([]types.Operation, 4)
	for i, role := range []common.Hash{proposerRole, bypasserRole, cancellerRole, executorRole} {
		data, perr := timelockAbi.Pack("grantRole", role, mcmC.Address())
		require.NoError(t, perr)

		operations[i] = types.Operation{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewOperation(
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

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client())}

	// Construct executor
	signable, err := NewSignable(&proposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	// Sign the hash with all signers
	for _, s := range sim.Signers {
		_, err = signable.SignAndAppend(NewPrivateKeySigner(s.PrivateKey))
		require.NoError(t, err)
	}

	// Sign with the invalid signer
	_, err = signable.SignAndAppend(NewPrivateKeySigner(invalidKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures()
	require.Error(t, err)
	// TODO: This should be an InvalidSignatureError, but right now the error is untyped. Depends on the import issue
	// require.IsType(t, &InvalidSignatureError{}, err)
	require.False(t, quorumMet)
}

func Test_Signable_Sign(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.HexToECDSA(testPrivateKeyHex)
	require.NoError(t, err)

	proposal := &Proposal{
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
					MCMAddress:      "0x01",
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: evm.NewOperation(
					common.HexToAddress("0x02"),
					[]byte("0x0000000"), // Use some random data since it doesn't matter
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	tests := []struct {
		name         string
		giveProposal *Proposal
		giveSigner   signer
		want         types.Signature
		wantErr      string
	}{
		{
			name:         "success: signs the proposal",
			giveProposal: proposal,
			giveSigner:   NewPrivateKeySigner(privKey),
			want: types.Signature{
				R: common.HexToHash("0x859c780e5df453945171c96f271c16b5baeeb6eadfa790d4e4d32ee72607334b"),
				S: common.HexToHash("0x3fd6128a489e81ecce6192804ea26ceaf542ae11f20caae65e6b65662f882eb4"),
				V: 0,
			},
		},
		{
			name:         "failure: invalid proposal",
			giveProposal: &Proposal{},
			giveSigner:   NewPrivateKeySigner(privKey),
			wantErr:      "Key: 'Proposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag\nKey: 'Proposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'required' tag\nKey: 'Proposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag\nKey: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag\nKey: 'Proposal.Operations' Error:Field validation for 'Operations' failed on the 'required' tag",
		},
		{
			name:         "failure: could not sign",
			giveProposal: proposal,
			giveSigner:   newFakeSigner([]byte{}, assert.AnError),
			wantErr:      assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// We need some inspectors to satisfy dependency validation, but this mock is unused.
			inspectors := map[types.ChainSelector]sdk.Inspector{
				chaintest.Chain1Selector: mocks.NewInspector(t),
			}

			// Ensure that there are no signatures to being with
			require.Empty(t, tt.giveProposal.Signatures)

			signable, err := NewSignable(tt.giveProposal, inspectors)
			require.NoError(t, err)

			got, err := signable.Sign(tt.giveSigner)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_SignAndAppend(t *testing.T) {
	t.Parallel()

	privKey, err := crypto.HexToECDSA(testPrivateKeyHex)
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
					MCMAddress:      "0x01",
				},
			},
		},
		Operations: []types.Operation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transaction: evm.NewOperation(
					common.HexToAddress("0x02"),
					[]byte("0x0000000"), // Use some random data since it doesn't matter
					big.NewInt(0),
					"RBACTimelock",
					[]string{"RBACTimelock", "GrantRole"},
				),
			},
		},
	}

	tests := []struct {
		name    string
		give    Proposal
		want    []types.Signature
		wantErr string
	}{
		{
			name: "success: signs the proposal",
			give: proposal,
			want: []types.Signature{
				{
					R: common.HexToHash("0x859c780e5df453945171c96f271c16b5baeeb6eadfa790d4e4d32ee72607334b"),
					S: common.HexToHash("0x3fd6128a489e81ecce6192804ea26ceaf542ae11f20caae65e6b65662f882eb4"),
					V: 0,
				},
			},
		},
		{
			name:    "failure: invalid proposal",
			give:    Proposal{},
			wantErr: "Key: 'Proposal.BaseProposal.Version' Error:Field validation for 'Version' failed on the 'required' tag\nKey: 'Proposal.BaseProposal.Kind' Error:Field validation for 'Kind' failed on the 'required' tag\nKey: 'Proposal.BaseProposal.ValidUntil' Error:Field validation for 'ValidUntil' failed on the 'required' tag\nKey: 'Proposal.BaseProposal.ChainMetadata' Error:Field validation for 'ChainMetadata' failed on the 'required' tag\nKey: 'Proposal.Operations' Error:Field validation for 'Operations' failed on the 'required' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector := mocks.NewInspector(t)
			inspectors := map[types.ChainSelector]sdk.Inspector{
				chaintest.Chain1Selector: inspector,
			}

			// Ensure that there are no signatures to being with
			require.Empty(t, tt.give.Signatures)

			signable, err := NewSignable(&tt.give, inspectors)
			require.NoError(t, err)
			require.NotNil(t, signable)

			got, err := signable.SignAndAppend(NewPrivateKeySigner(privKey))

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)

				// Ensure that the signature was appended
				require.Len(t, tt.give.Signatures, len(tt.want))
				require.ElementsMatch(t, tt.want, tt.give.Signatures)
				require.Contains(t, tt.give.Signatures, got)
			}
		})
	}
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
						chaintest.Chain2Selector: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(config1, nil)
				m.inspector2.EXPECT().GetConfig("0x02").Return(config2, nil)

				return map[types.ChainSelector]sdk.Inspector{
					chaintest.Chain1Selector: m.inspector1,
					chaintest.Chain2Selector: m.inspector2,
				}
			},
			want: map[types.ChainSelector]*types.Config{
				chaintest.Chain1Selector: config1,
				chaintest.Chain2Selector: config2,
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(nil, assert.AnError)

				return map[types.ChainSelector]sdk.Inspector{
					chaintest.Chain1Selector: m.inspector1,
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

func Test_Signable_Simulate(t *testing.T) {
	t.Parallel()

	operations := []types.Operation{
		{
			ChainSelector: chaintest.Chain1Selector,
			Transaction: evm.NewOperation(
				common.HexToAddress("0x02"),
				[]byte("0x0000000"), // Use some random data since it doesn't matter
				big.NewInt(0),
				"RBACTimelock",
				[]string{"RBACTimelock", "GrantRole"},
			),
		},
		{
			ChainSelector: chaintest.Chain2Selector,
			Transaction: evm.NewOperation(
				common.HexToAddress("0x02"),
				[]byte("0x0000000"), // Use some random data since it doesn't matter
				big.NewInt(0),
				"RBACTimelock",
				[]string{"RBACTimelock", "GrantRole"},
			),
		},
	}

	tests := []struct {
		name           string
		give           Proposal
		giveSimulators func(*simulatorMocks) map[types.ChainSelector]sdk.Simulator
		wantErr        string
	}{
		{
			name: "success",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
						chaintest.Chain2Selector: {MCMAddress: "0x02"},
					},
				},
				Operations: operations,
			},
			giveSimulators: func(m *simulatorMocks) map[types.ChainSelector]sdk.Simulator {
				m.simulator1.EXPECT().SimulateOperation(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				m.simulator2.EXPECT().SimulateOperation(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return map[types.ChainSelector]sdk.Simulator{
					chaintest.Chain1Selector: m.simulator1,
					chaintest.Chain2Selector: m.simulator2,
				}
			},
		},
		{
			name: "failure: no simulators",
			give: Proposal{},
			giveSimulators: func(m *simulatorMocks) map[types.ChainSelector]sdk.Simulator {
				return nil
			},
			wantErr: ErrSimulatorsNotProvided.Error(),
		},
		{
			name: "failure: simulator not found",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
					},
				},
				Operations: operations,
			},
			giveSimulators: func(m *simulatorMocks) map[types.ChainSelector]sdk.Simulator {
				return map[types.ChainSelector]sdk.Simulator{}
			},
			wantErr: "simulator not found for chain 3379446385462418246",
		},
		{
			name: "failure: on chain simulate failure",
			give: Proposal{
				BaseProposal: BaseProposal{
					ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
					},
				},
				Operations: operations,
			},
			giveSimulators: func(m *simulatorMocks) map[types.ChainSelector]sdk.Simulator {
				m.simulator1.EXPECT().SimulateOperation(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
				return map[types.ChainSelector]sdk.Simulator{
					chaintest.Chain1Selector: m.simulator1,
					chaintest.Chain2Selector: m.simulator2,
				}
			},
			wantErr: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			simulator1 := sdkmocks.NewSimulator(t)
			simulator2 := sdkmocks.NewSimulator(t)

			giveSimulators := tt.giveSimulators(&simulatorMocks{
				simulator1: simulator1,
				simulator2: simulator2,
			})

			signable := &Signable{
				proposal:   &tt.give,
				simulators: giveSimulators,
			}

			err := signable.Simulate()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
						chaintest.Chain2Selector: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(config1, nil)
				m.inspector2.EXPECT().GetConfig("0x02").Return(config2, nil)

				return map[types.ChainSelector]sdk.Inspector{
					chaintest.Chain1Selector: m.inspector1,
					chaintest.Chain2Selector: m.inspector2,
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
						chaintest.Chain2Selector: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetConfig("0x01").Return(config1, nil)
				m.inspector2.EXPECT().GetConfig("0x02").Return(config3, nil)

				return map[types.ChainSelector]sdk.Inspector{
					chaintest.Chain1Selector: m.inspector1,
					chaintest.Chain2Selector: m.inspector2,
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
						chaintest.Chain2Selector: {MCMAddress: "0x02"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetOpCount("0x01").Return(100, nil)
				m.inspector2.EXPECT().GetOpCount("0x02").Return(200, nil)

				return map[types.ChainSelector]sdk.Inspector{
					chaintest.Chain1Selector: m.inspector1,
					chaintest.Chain2Selector: m.inspector2,
				}
			},
			want: map[types.ChainSelector]uint64{
				chaintest.Chain1Selector: 100,
				chaintest.Chain2Selector: 200,
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
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
						chaintest.Chain1Selector: {MCMAddress: "0x01"},
					},
				},
			},
			giveInspectors: func(m *inspectorMocks) map[types.ChainSelector]sdk.Inspector {
				m.inspector1.EXPECT().GetOpCount("0x01").Return(0, assert.AnError)

				return map[types.ChainSelector]sdk.Inspector{
					chaintest.Chain1Selector: m.inspector1,
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
