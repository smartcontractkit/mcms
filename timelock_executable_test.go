package mcms

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	geth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	"github.com/smartcontractkit/mcms/types"
)

var (
	proposerRole  = crypto.Keccak256Hash([]byte("PROPOSER_ROLE"))
	bypasserRole  = crypto.Keccak256Hash([]byte("BYPASSER_ROLE"))
	cancellerRole = crypto.Keccak256Hash([]byte("CANCELLER_ROLE"))
	adminRole     = crypto.Keccak256Hash([]byte("ADMIN_ROLE"))
)

func Test_NewTimelockExecutable(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()

		executor = mocks.NewTimelockExecutor(t)

		chainMetadata = map[types.ChainSelector]types.ChainMetadata{
			chaintest.Chain1Selector: {
				StartingOpCount: 1,
				MCMAddress:      "0x1234",
			},
		}

		chainMetadataBad = map[types.ChainSelector]types.ChainMetadata{
			types.ChainSelector(1): {
				StartingOpCount: 1,
				MCMAddress:      "0x1234",
			},
		}

		timelockAddresses = map[types.ChainSelector]string{
			chaintest.Chain1Selector: "0x1234",
		}

		tx = types.Transaction{
			To:               "0x1234",
			AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
			Data:             common.Hex2Bytes("0x0"),
			OperationMetadata: types.OperationMetadata{
				ContractType: "Test contract",
				Tags:         []string{"testTag1", "testTag2"},
			},
		}

		batchOps = []types.BatchOperation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{
					tx,
				},
			},
		}
	)

	tests := []struct {
		name          string
		giveProposal  *TimelockProposal
		giveExecutors map[types.ChainSelector]sdk.TimelockExecutor
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name: "success",
			giveProposal: &TimelockProposal{
				BaseProposal: BaseProposal{
					Version:              "v1",
					Kind:                 types.KindTimelockProposal,
					Description:          "description",
					ValidUntil:           2004259681,
					OverridePreviousRoot: false,
					Signatures:           []types.Signature{},
					ChainMetadata:        chainMetadata,
				},
				Action:            types.TimelockActionSchedule,
				Delay:             types.MustParseDuration("1h"),
				TimelockAddresses: timelockAddresses,
				Operations:        batchOps,
			},
			giveExecutors: map[types.ChainSelector]sdk.TimelockExecutor{
				chaintest.Chain1Selector: executor,
			},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "failure: converter from executor error",
			giveProposal: &TimelockProposal{
				BaseProposal: BaseProposal{
					Version:              "v1",
					Kind:                 types.KindTimelockProposal,
					Description:          "description",
					ValidUntil:           2004259681,
					OverridePreviousRoot: false,
					Signatures:           []types.Signature{},
					ChainMetadata:        chainMetadataBad,
				},
				Action:            types.TimelockActionSchedule,
				Delay:             types.MustParseDuration("1h"),
				TimelockAddresses: timelockAddresses,
				Operations:        batchOps,
			},
			giveExecutors: map[types.ChainSelector]sdk.TimelockExecutor{
				types.ChainSelector(1): executor,
			},
			wantErr:    true,
			wantErrMsg: "unable to set predecessors: unable to create converter from executor: chain family not found for selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTimelockExecutable(ctx, tt.giveProposal, tt.giveExecutors)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_TimelockExecutable_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	defaultProposal := func() *TimelockProposal {
		return &TimelockProposal{
			BaseProposal: BaseProposal{
				Version:              "v1",
				Kind:                 types.KindTimelockProposal,
				Description:          "description",
				ValidUntil:           2004259681,
				OverridePreviousRoot: false,
				Signatures:           []types.Signature{},
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: {
						StartingOpCount: 1,
						MCMAddress:      "0x1234",
					},
				},
			},
			Action:            types.TimelockActionSchedule,
			Delay:             types.MustParseDuration("1h"),
			TimelockAddresses: map[types.ChainSelector]string{chaintest.Chain1Selector: "0x5678"},
			Operations: []types.BatchOperation{{
				ChainSelector: chaintest.Chain1Selector,
				Transactions: []types.Transaction{{
					To:               "0x9012",
					AdditionalFields: json.RawMessage([]byte(`{"value": 0}`)),
					Data:             common.Hex2Bytes("0x0"),
					OperationMetadata: types.OperationMetadata{
						ContractType: "Test contract",
						Tags:         []string{"testTag1", "testTag2"},
					},
				}},
			}},
		}
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor)
		index   int
		want    string
		wantErr string
		option  Option
	}{
		{
			name: "success",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()

				executor := mocks.NewTimelockExecutor(t)
				executor.EXPECT().
					Execute(ctx, mock.Anything, "0x5678", mock.Anything, mock.Anything).
					Return(types.TransactionResult{
						Hash:        "signature",
						ChainFamily: chain_selectors.FamilyEVM,
					}, nil).Once()
				executors := map[types.ChainSelector]sdk.TimelockExecutor{chaintest.Chain1Selector: executor}

				return defaultProposal(), executors
			},
			want: "signature",
		},
		{
			name: "success with callproxy",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()

				executor := mocks.NewTimelockExecutor(t)
				executor.EXPECT().
					Execute(ctx, mock.Anything, "0xABCD", mock.Anything, mock.Anything).
					Return(types.TransactionResult{
						Hash:        "signature",
						ChainFamily: chain_selectors.FamilyEVM,
					}, nil).Once()
				executors := map[types.ChainSelector]sdk.TimelockExecutor{chaintest.Chain1Selector: executor}

				return defaultProposal(), executors
			},
			option: WithCallProxy("0xABCD"),
			want:   "signature",
		},
		{
			name: "failure: execute error",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()

				executor := mocks.NewTimelockExecutor(t)
				executor.EXPECT().
					Execute(ctx, mock.Anything, "0x5678", mock.Anything, mock.Anything).
					Return(types.TransactionResult{}, fmt.Errorf("execute error")).Once()
				executors := map[types.ChainSelector]sdk.TimelockExecutor{chaintest.Chain1Selector: executor}

				return defaultProposal(), executors
			},
			wantErr: "execute error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposal, executors := tt.setup(t)
			timelockExecutable, err := NewTimelockExecutable(ctx, proposal, executors)
			require.NoError(t, err)

			var got types.TransactionResult
			if tt.option != nil {
				got, err = timelockExecutable.Execute(ctx, tt.index, tt.option)
			} else {
				got, err = timelockExecutable.Execute(ctx, tt.index)
			}

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got.Hash)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_ScheduleAndExecuteProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		targetRoles []common.Hash
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "valid schedule and execute proposal with one tx and one op",
			targetRoles: []common.Hash{proposerRole},
			wantErr:     false,
			wantErrMsg:  "",
		},
		{
			name:        "valid schedule and execute proposal with one tx and three ops",
			targetRoles: []common.Hash{proposerRole, bypasserRole, cancellerRole},
			wantErr:     false,
			wantErrMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheduleAndExecuteGrantRolesProposal(t, tt.targetRoles)
		})
	}
}

func Test_ScheduleAndCancelProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		targetRoles []common.Hash
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "valid schedule and cancel proposal with one tx and one op",
			targetRoles: []common.Hash{proposerRole},
			wantErr:     false,
			wantErrMsg:  "",
		},
		{
			name:        "valid schedule and cancel proposal with one tx and three ops",
			targetRoles: []common.Hash{proposerRole, bypasserRole, cancellerRole},
			wantErr:     false,
			wantErrMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheduleAndCancelGrantRolesProposal(t, tt.targetRoles)
		})
	}
}

func scheduleGrantRolesProposal(
	t *testing.T, targetRoles []common.Hash, delay types.Duration) (evmsim.SimulatedChain, *bindings.ManyChainMultiSig,
	*bindings.RBACTimelock, TimelockProposal, []common.Hash,
) {
	t.Helper()

	sim := evmsim.NewSimulatedChain(t, 1)
	mcmC, _ := sim.DeployMCMContract(t, sim.Signers[0])
	sim.SetMCMSConfig(t, sim.Signers[0], mcmC)

	// Deploy a timelock contract for testing
	timelockC, _ := sim.DeployRBACTimelock(t, sim.Signers[0], sim.Signers[0].Address(t),
		[]common.Address{mcmC.Address()},
		[]common.Address{mcmC.Address(), sim.Signers[0].Address(t)},
		[]common.Address{mcmC.Address()},
		[]common.Address{mcmC.Address()},
	)

	// Give timelock admin permissions
	_, err := timelockC.GrantRole(sim.Signers[0].NewTransactOpts(t), adminRole, timelockC.Address())
	require.NoError(t, err)
	sim.Backend.Commit()

	// renounce admin role
	_, err = timelockC.RenounceRole(sim.Signers[0].NewTransactOpts(t), adminRole, sim.Signers[0].Address(t))
	require.NoError(t, err)
	sim.Backend.Commit()

	// Construct example transactions
	grantRoleDatas := make([][]byte, 0)
	timelockAbi, err := bindings.RBACTimelockMetaData.GetAbi()
	var grantRoleData []byte
	for _, role := range targetRoles {
		require.NoError(t, err)
		grantRoleData, err = timelockAbi.Pack("grantRole", role, sim.Signers[0].Address(t))
		require.NoError(t, err)
		grantRoleDatas = append(grantRoleDatas, grantRoleData)
	}

	// Validate Contract State and verify role does not exist
	var hasRole bool
	for _, role := range targetRoles {
		hasRole, err = timelockC.HasRole(&bind.CallOpts{}, role, sim.Signers[0].Address(t))
		require.NoError(t, err)
		require.False(t, hasRole)
	}

	// Construct transactions
	transactions := make([]types.Transaction, 0)
	for _, data := range grantRoleDatas {
		transactions = append(transactions, evm.NewTransaction(
			timelockC.Address(),
			data,
			big.NewInt(0),
			"RBACTimelock",
			[]string{"RBACTimelock", "GrantRole"},
		))
	}

	// Construct a proposal
	proposal := TimelockProposal{
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
		Operations: []types.BatchOperation{
			{
				ChainSelector: chaintest.Chain1Selector,
				Transactions:  transactions,
			},
		},
		Action: types.TimelockActionSchedule,
		Delay:  delay,
		TimelockAddresses: map[types.ChainSelector]string{
			chaintest.Chain1Selector: timelockC.Address().Hex(),
		},
	}

	return sim, mcmC, timelockC, proposal, targetRoles
}

func scheduleAndExecuteGrantRolesProposal(t *testing.T, targetRoles []common.Hash) {
	t.Helper()
	ctx := t.Context()

	sim, mcmC, timelockC, proposal, _ := scheduleGrantRolesProposal(t, targetRoles, types.MustParseDuration("5s"))

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		chaintest.Chain1Selector: &evm.TimelockConverter{},
	}

	// convert proposal to mcms
	mcmsProposal, predecessors, err := proposal.Convert(ctx, converters)
	require.NoError(t, err)
	mcmsProposal.UseSimulatedBackend(true)
	tree, err := mcmsProposal.MerkleTree()
	require.NoError(t, err)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{
		chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client()),
	}

	// Construct executor
	signable, err := NewSignable(&mcmsProposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := mcmsProposal.GetEncoders()
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
	executable, err := NewExecutable(&mcmsProposal, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err := executable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotEmpty(t, tx.Hash)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(tree.Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	var receipt *geth_types.Receipt
	for i := range proposal.Operations {
		tx, err = executable.Execute(ctx, i)
		require.NoError(t, err)
		require.NotEmpty(t, tx.Hash)
		sim.Backend.Commit()

		// Wait for the transaction to be mined
		receipt, err = testutils.WaitMinedWithTxHash(ctx, sim.Backend.Client(), common.HexToHash(tx.Hash))
		require.NoError(t, err)
		require.NotNil(t, receipt)
		require.Equal(t, geth_types.ReceiptStatusSuccessful, receipt.Status)
	}

	// Check the state of the MCMS contract
	newOpCount, err := mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(1), newOpCount.Uint64())

	// Construct executors
	tExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		chaintest.Chain1Selector: evm.NewTimelockExecutor(
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Create new executable
	tExecutable, err := NewTimelockExecutable(ctx, &proposal, tExecutors)
	require.NoError(t, err)

	for i := range predecessors {
		if i == 0 || predecessors[i] == ZeroHash {
			continue
		}
		var isOperation, isOperationPending, isOperationReady bool
		isOperation, err = timelockC.IsOperation(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.True(t, isOperation)
		isOperationPending, err = timelockC.IsOperationPending(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.True(t, isOperationPending)
		isOperationReady, err = timelockC.IsOperationReady(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.False(t, isOperationReady)
	}

	opIdx := 0
	requireOperationPending(t, ctx, tExecutable, &proposal, opIdx)
	requireOperationNotReady(t, ctx, tExecutable, &proposal, opIdx)
	requireOperationNotDone(t, ctx, tExecutable, &proposal, opIdx)

	// sleep for 5 seconds and then mine a block
	require.NoError(t, sim.Backend.AdjustTime(5*time.Second))
	sim.Backend.Commit() // Note < 1.14 geth needs a commit after adjusting time.

	requireOperationPending(t, ctx, tExecutable, &proposal, opIdx)
	requireOperationReady(t, ctx, tExecutable, &proposal, opIdx)
	requireOperationNotDone(t, ctx, tExecutable, &proposal, opIdx)

	// Execute the proposal
	_, err = tExecutable.Execute(ctx, opIdx)
	require.NoError(t, err)
	sim.Backend.Commit()

	requireOperationNotPending(t, ctx, tExecutable, &proposal, opIdx)
	requireOperationNotReady(t, ctx, tExecutable, &proposal, opIdx)
	requireOperationDone(t, ctx, tExecutable, &proposal, opIdx)

	// Check the state of the timelock contract
	for _, role := range targetRoles {
		roleCount, err := timelockC.GetRoleMemberCount(&bind.CallOpts{}, role)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(2), roleCount)
		newRoleOwner, err := timelockC.GetRoleMember(&bind.CallOpts{}, role, big.NewInt(1))
		require.NoError(t, err)
		require.Equal(t, sim.Signers[0].Address(t).Hex(), newRoleOwner.Hex())
	}
}

func scheduleAndCancelGrantRolesProposal(t *testing.T, targetRoles []common.Hash) {
	t.Helper()
	ctx := t.Context()

	sim, mcmC, timelockC, proposal, _ := scheduleGrantRolesProposal(t, targetRoles, types.MustParseDuration("5m"))

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		chaintest.Chain1Selector: &evm.TimelockConverter{},
	}

	// convert proposal to mcms
	mcmsProposal, predecessors, err := proposal.Convert(ctx, converters)
	require.NoError(t, err)
	mcmsProposal.UseSimulatedBackend(true)
	tree, err := mcmsProposal.MerkleTree()
	require.NoError(t, err)

	// Gen caller map for easy access
	inspectors := map[types.ChainSelector]sdk.Inspector{
		chaintest.Chain1Selector: evm.NewInspector(sim.Backend.Client()),
	}

	// Construct executor
	signable, err := NewSignable(&mcmsProposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, signable)

	_, err = signable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet)

	// Construct encoders
	encoders, err := mcmsProposal.GetEncoders()
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
	executable, err := NewExecutable(&mcmsProposal, executors)
	require.NoError(t, err)

	// SetRoot on the contract
	txHash, err := executable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)
	sim.Backend.Commit()

	// Validate Contract State and verify root was set
	root, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, root.Root, [32]byte(tree.Root.Bytes()))
	require.Equal(t, root.ValidUntil, proposal.ValidUntil)

	// Execute the proposal
	var receipt *geth_types.Receipt
	var tx types.TransactionResult
	for i := range proposal.Operations {
		tx, err = executable.Execute(ctx, i)
		require.NoError(t, err)
		require.NotEmpty(t, txHash)
		sim.Backend.Commit()

		// Wait for the transaction to be mined
		receipt, err = testutils.WaitMinedWithTxHash(ctx, sim.Backend.Client(), common.HexToHash(tx.Hash))
		require.NoError(t, err)
		require.NotNil(t, receipt)
		require.Equal(t, geth_types.ReceiptStatusSuccessful, receipt.Status)
	}

	// Check the state of the MCMS contract
	newOpCount, err := mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(1), newOpCount.Uint64())

	// Construct executors
	tExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		chaintest.Chain1Selector: evm.NewTimelockExecutor(
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Create new executable
	tExecutable, err := NewTimelockExecutable(ctx, &proposal, tExecutors)
	require.NoError(t, err)

	for i := range predecessors {
		if i == 0 || predecessors[i] == ZeroHash {
			continue
		}

		var isOperation, isOperationPending, isOperationReady bool
		isOperation, err = timelockC.IsOperation(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.True(t, isOperation)
		isOperationPending, err = timelockC.IsOperationPending(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.True(t, isOperationPending)
		isOperationReady, err = timelockC.IsOperationReady(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.False(t, isOperationReady)
	}

	// Check IsReady function fails
	err = tExecutable.IsReady(ctx)
	require.Error(t, err)

	// Construct cancel proposal
	cancelProposal := proposal
	cancelProposal.ChainMetadata[chaintest.Chain1Selector] = types.ChainMetadata{
		StartingOpCount: 1,
		MCMAddress:      mcmC.Address().Hex(),
	}
	cancelProposal.Action = types.TimelockActionCancel

	// TODO: in practice ValidUntil would need to be updated here, but we set a validUntil
	// far enough in the future that it doesn't matter for this test.

	// convert cancelProposal to mcms
	cancelMcmsProposal, _, err := cancelProposal.Convert(ctx, converters)
	require.NoError(t, err)
	cancelMcmsProposal.UseSimulatedBackend(true)
	// cancelTree, err := cancelMcmsProposal.MerkleTree()
	require.NoError(t, err)

	// Construct executor
	cancelSignable, err := NewSignable(&cancelMcmsProposal, inspectors)
	require.NoError(t, err)
	require.NotNil(t, cancelSignable)

	_, err = cancelSignable.SignAndAppend(NewPrivateKeySigner(sim.Signers[0].PrivateKey))
	require.NoError(t, err)

	// Validate the signatures
	cancelQuorumMet, err := cancelSignable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, cancelQuorumMet)

	// Construct encoders
	cancelEncoders, err := cancelMcmsProposal.GetEncoders()
	require.NoError(t, err)

	// Construct executors
	cancelExecutors := map[types.ChainSelector]sdk.Executor{
		chaintest.Chain1Selector: evm.NewExecutor(
			cancelEncoders[chaintest.Chain1Selector].(*evm.Encoder),
			sim.Backend.Client(),
			sim.Signers[0].NewTransactOpts(t),
		),
	}

	// Construct executable
	cancelExecutable, err := NewExecutable(&cancelMcmsProposal, cancelExecutors)
	require.NoError(t, err)

	// SetRoot on the contract
	tx, err = cancelExecutable.SetRoot(ctx, chaintest.Chain1Selector)
	require.NoError(t, err)
	require.NotEmpty(t, txHash)
	sim.Backend.Commit()

	cancelReceipt, err := testutils.WaitMinedWithTxHash(ctx, sim.Backend.Client(), common.HexToHash(tx.Hash))
	require.NoError(t, err)
	require.NotNil(t, cancelReceipt)
	require.Equal(t, geth_types.ReceiptStatusSuccessful, cancelReceipt.Status)

	// Validate Contract State and verify root was set
	cancelRoot, err := mcmC.GetRoot(&bind.CallOpts{})
	require.NoError(t, err)
	// require.Equal(t, cancelRoot.Root, [32]byte(cancelTree.Root.Bytes()))
	require.Equal(t, cancelRoot.ValidUntil, cancelProposal.ValidUntil)

	// Execute the cancelProposal
	for i := range cancelProposal.Operations {
		tx, err = cancelExecutable.Execute(ctx, i)
		require.NoError(t, err)
		require.NotEmpty(t, txHash)
		sim.Backend.Commit()

		// Wait for the transaction to be mined
		cancelReceipt, err = testutils.WaitMinedWithTxHash(ctx, sim.Backend.Client(), common.HexToHash(tx.Hash))
		require.NoError(t, err)
		require.NotNil(t, cancelReceipt)
		require.Equal(t, geth_types.ReceiptStatusSuccessful, cancelReceipt.Status)
	}

	// Check the state of the MCMS contract
	newOpCount, err = mcmC.GetOpCount(&bind.CallOpts{})
	require.NoError(t, err)
	require.NotNil(t, newOpCount)
	require.Equal(t, uint64(2), newOpCount.Uint64())

	for i := range predecessors {
		if i == 0 {
			continue
		}

		var isOperation, isOperationPending, isOperationReady bool
		isOperation, err = timelockC.IsOperation(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.False(t, isOperation)
		isOperationPending, err = timelockC.IsOperationPending(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.False(t, isOperationPending)
		isOperationReady, err = timelockC.IsOperationReady(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.False(t, isOperationReady)
	}
}

func Test_TimelockExecutable_GetChainSpecificIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// We'll create a helper function for generating a minimal proposal
	// that won't fail validation. It includes multiple operations across
	// different chain selectors.
	defaultProposal := func() *TimelockProposal {
		// A minimal transaction for each operation
		tx := types.Transaction{
			To:               "0x1234",
			AdditionalFields: json.RawMessage([]byte(`{"value":0}`)),
			Data:             common.Hex2Bytes("0x"),
			OperationMetadata: types.OperationMetadata{
				ContractType: "TestContract",
				Tags:         []string{"tag1"},
			},
		}

		// We'll set up 6 operations across 3 different chain selectors
		operations := []types.BatchOperation{
			{ // index 0 => chain1
				ChainSelector: chaintest.Chain1Selector,
				Transactions:  []types.Transaction{tx},
			},
			{ // index 1 => chain2
				ChainSelector: chaintest.Chain2Selector,
				Transactions:  []types.Transaction{tx},
			},
			{ // index 2 => chain1
				ChainSelector: chaintest.Chain1Selector,
				Transactions:  []types.Transaction{tx},
			},
			{ // index 3 => chain3
				ChainSelector: chaintest.Chain3Selector,
				Transactions:  []types.Transaction{tx},
			},
			{ // index 4 => chain1
				ChainSelector: chaintest.Chain1Selector,
				Transactions:  []types.Transaction{tx},
			},
			{ // index 5 => chain2
				ChainSelector: chaintest.Chain2Selector,
				Transactions:  []types.Transaction{tx},
			},
		}

		return &TimelockProposal{
			BaseProposal: BaseProposal{
				Version:    "v1",
				Kind:       types.KindTimelockProposal, // Must match for NewTimelockExecutable
				ValidUntil: 2004259681,                 // Some future time
				ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
					chaintest.Chain1Selector: {
						StartingOpCount: 0,
						MCMAddress:      "0xAAAA",
					},
					chaintest.Chain2Selector: {
						StartingOpCount: 0,
						MCMAddress:      "0xBBBB",
					},
					chaintest.Chain3Selector: {
						StartingOpCount: 0,
						MCMAddress:      "0xCCCC",
					},
				},
				Signatures: []types.Signature{},
			},
			Action: types.TimelockActionSchedule, // Must be "schedule" for NewTimelockExecutable
			Delay:  types.MustParseDuration("1h"),
			TimelockAddresses: map[types.ChainSelector]string{
				chaintest.Chain1Selector: "0x1111",
				chaintest.Chain2Selector: "0x2222",
				chaintest.Chain3Selector: "0x3333",
			},
			Operations: operations,
		}
	}

	t.Run("chain-specific indexing across multiple chains", func(t *testing.T) {
		t.Parallel()

		proposal := defaultProposal()

		// We don't actually need executors to test GetChainSpecificIndex,
		// so we'll pass an empty map. (Or nil is fine too.)
		executors := map[types.ChainSelector]sdk.TimelockExecutor{}

		// Create TimelockExecutable
		tlExecutable, err := NewTimelockExecutable(ctx, proposal, executors)
		require.NoError(t, err)

		// Each test-case checks the chain-specific index for a given global index
		testCases := []struct {
			name        string
			globalIndex int
			want        int
		}{
			{
				name:        "0th op, chain1 => 1st on that chain",
				globalIndex: 0,
				want:        1,
			},
			{
				name:        "1st op, chain2 => 1st on that chain",
				globalIndex: 1,
				want:        1,
			},
			{
				name:        "2nd op, chain1 => 2nd on that chain",
				globalIndex: 2,
				want:        2,
			},
			{
				name:        "3rd op, chain3 => 1st on that chain",
				globalIndex: 3,
				want:        1,
			},
			{
				name:        "4th op, chain1 => 3rd on that chain",
				globalIndex: 4,
				want:        3,
			},
			{
				name:        "5th op, chain2 => 2nd on that chain",
				globalIndex: 5,
				want:        2,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := tlExecutable.GetChainSpecificIndex(tc.globalIndex)
				require.Equal(t, tc.want, got)
			})
		}
	})
}

func requireOperationPending(
	t *testing.T, ctx context.Context, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int,
) {
	t.Helper()
	err := tExecutable.IsOperationPending(ctx, opIdx)
	require.NoError(t, err)
	for chainSelector := range proposal.ChainMetadata {
		err = tExecutable.IsChainPending(ctx, chainSelector)
		require.NoError(t, err)
	}
}

func requireOperationNotPending(
	t *testing.T, ctx context.Context, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int,
) {
	t.Helper()
	err := tExecutable.IsOperationPending(ctx, opIdx)
	require.ErrorContains(t, err, "operation 0 is not pending")
	for chainSelector := range proposal.ChainMetadata {
		err = tExecutable.IsChainPending(ctx, chainSelector)
		require.ErrorContains(t, err, "operation 0 is not pending")
	}
}

func requireOperationReady(
	t *testing.T, ctx context.Context, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int,
) {
	t.Helper()
	err := tExecutable.IsOperationReady(ctx, opIdx)
	require.NoError(t, err)
	for chainSelector := range proposal.ChainMetadata {
		err = tExecutable.IsChainReady(ctx, chainSelector)
		require.NoError(t, err)
	}
}

func requireOperationNotReady(
	t *testing.T, ctx context.Context, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int,
) {
	t.Helper()
	err := tExecutable.IsOperationReady(ctx, opIdx)
	require.ErrorContains(t, err, "operation 0 is not ready")
	for chainSelector := range proposal.ChainMetadata {
		err = tExecutable.IsChainReady(ctx, chainSelector)
		require.ErrorContains(t, err, "operation 0 is not ready")
	}
}

func requireOperationDone(
	t *testing.T, ctx context.Context, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int,
) {
	t.Helper()
	err := tExecutable.IsOperationDone(ctx, opIdx)
	require.NoError(t, err)
	for chainSelector := range proposal.ChainMetadata {
		err = tExecutable.IsChainDone(ctx, chainSelector)
		require.NoError(t, err)
	}
}

func requireOperationNotDone(
	t *testing.T, ctx context.Context, tExecutable *TimelockExecutable, proposal *TimelockProposal, opIdx int,
) {
	t.Helper()
	err := tExecutable.IsOperationDone(ctx, opIdx)
	require.ErrorContains(t, err, "operation 0 is not done")
	for chainSelector := range proposal.ChainMetadata {
		err = tExecutable.IsChainDone(ctx, chainSelector)
		require.ErrorContains(t, err, "operation 0 is not done")
	}
}
