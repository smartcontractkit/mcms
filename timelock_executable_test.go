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
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/internal/testutils/evmsim"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/mocks"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
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
		executor = mocks.NewTimelockExecutor(t)

		chainMetadata = map[types.ChainSelector]types.ChainMetadata{
			chaintest.Chain1Selector: {
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
				types.ChainSelector(1): executor,
			},
			wantErr:    false,
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTimelockExecutable(tt.giveProposal, tt.giveExecutors)

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
	}{
		{
			name: "success",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()

				executor := mocks.NewTimelockExecutor(t)
				executor.EXPECT().
					Execute(ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("signature", nil).Once()
				executors := map[types.ChainSelector]sdk.TimelockExecutor{chaintest.Chain1Selector: executor}

				return defaultProposal(), executors
			},
			want: "signature",
		},
		{
			name: "failure: converter from executor error",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()
				executors := map[types.ChainSelector]sdk.TimelockExecutor{
					types.ChainSelector(12345789): mocks.NewTimelockExecutor(t),
				}

				return defaultProposal(), executors
			},
			wantErr: "unable to set predecessors: unable to create converter from executor: chain family not found for selector",
		},
		{
			name: "failure: convert error",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()

				solanaClient := &rpc.Client{}
				solanaAuth, err := solana.NewRandomPrivateKey()
				require.NoError(t, err)
				executors := map[types.ChainSelector]sdk.TimelockExecutor{
					chaintest.Chain4Selector: solanasdk.NewTimelockExecutor(solanaClient, solanaAuth),
				}

				return defaultProposal(), executors
			},
			wantErr: "unable to set predecessors: unable to find converter for chain selector",
		},
		{
			name: "failure: execute error",
			setup: func(t *testing.T) (*TimelockProposal, map[types.ChainSelector]sdk.TimelockExecutor) {
				t.Helper()

				executor := mocks.NewTimelockExecutor(t)
				executor.EXPECT().
					Execute(ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return("", fmt.Errorf("execute error")).Once()
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
			timelockExecutable, err := NewTimelockExecutable(proposal, executors)
			require.NoError(t, err)

			got, err := timelockExecutable.Execute(ctx, tt.index, "")

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_ScheduleAndExecuteProposal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

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

			scheduleAndExecuteGrantRolesProposal(t, ctx, tt.targetRoles)
		})
	}
}

func scheduleAndExecuteGrantRolesProposal(t *testing.T, ctx context.Context, targetRoles []common.Hash) {
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
		transactions = append(transactions, evm.NewOperation(
			timelockC.Address(),
			data,
			big.NewInt(0),
			"RBACTimelock",
			[]string{"RBACTimelock", "GrantRole"},
		))
	}

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		chaintest.Chain1Selector: &evm.TimelockConverter{},
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
		Delay:  types.MustParseDuration("5s"),
		TimelockAddresses: map[types.ChainSelector]string{
			chaintest.Chain1Selector: timelockC.Address().Hex(),
		},
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
	for i := range proposal.Operations {
		txHash, err = executable.Execute(ctx, i)
		require.NoError(t, err)
		require.NotEmpty(t, txHash)
		sim.Backend.Commit()

		// Wait for the transaction to be mined
		receipt, err = testutils.WaitMinedWithTxHash(ctx, sim.Backend.Client(), common.HexToHash(txHash))
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
	tExecutable, err := NewTimelockExecutable(&proposal, tExecutors)
	require.NoError(t, err)

	for i := range predecessors {
		if i == 0 {
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

	// sleep for 5 seconds and then mine a block
	require.NoError(t, sim.Backend.AdjustTime(5*time.Second))
	sim.Backend.Commit() // Note < 1.14 geth needs a commit after adjusting time.

	// Check that the operation is now ready
	err = tExecutable.IsReady(ctx)
	require.NoError(t, err)

	// Execute the proposal
	_, err = tExecutable.Execute(ctx, 0, "")
	require.NoError(t, err)
	sim.Backend.Commit()

	// Check that the operation is done
	for i := range predecessors {
		if i == 0 {
			continue
		}

		isOperationDone, err := timelockC.IsOperationDone(&bind.CallOpts{}, predecessors[i])
		require.NoError(t, err)
		require.True(t, isOperationDone)
	}

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
