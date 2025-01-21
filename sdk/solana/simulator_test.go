package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewSimulator(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth := solana.MustPrivateKeyFromBase58("DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc")
	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)
	encoder := NewEncoder(chainSelector, 1, false)

	simulator, err := NewSimulator(client, auth, encoder)
	require.NoError(t, err)
	require.NotNil(t, simulator)
}

func TestSimulator_SimulateOperation(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx      context.Context
		metadata types.ChainMetadata
		op       types.Operation
	}
	selector := cselectors.SOLANA_DEVNET.Selector
	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)
	contractID := fmt.Sprintf("%s.%s", testMCMProgramID.String(), testPDASeed)

	ctx := context.Background()

	require.NoError(t, err)
	testWallet := solana.NewWallet()
	ix, err := system.NewTransferInstruction(20*solana.LAMPORTS_PER_SOL, auth.PublicKey(), testWallet.PublicKey()).ValidateAndBuild()
	require.NoError(t, err)
	data, err := ix.Data()
	require.NoError(t, err)
	tx, err := NewTransaction(solana.SystemProgramID.String(), data, ix.Accounts(), "solana-testing", []string{})
	tests := []struct {
		name string
		args args

		mockSetup func(*mocks.JSONRPCClient)
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{
			name: "success: SimulateOperation",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: contractID,
				},
				ctx: ctx,
				op: types.Operation{
					Transaction:   tx,
					ChainSelector: types.ChainSelector(selector),
				},
			},
			mockSetup: func(m *mocks.JSONRPCClient) {
				mockSolanaSimulation(t, m, 20, 5, "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", nil)
			},

			assertion: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jsonRPCClient := mocks.NewJSONRPCClient(t)
			encoder := NewEncoder(types.ChainSelector(selector), 0, false)
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			tt.mockSetup(jsonRPCClient)
			sim, err := NewSimulator(client, auth, encoder)
			require.NoError(t, err)
			err = sim.SimulateOperation(ctx, tt.args.metadata, tt.args.op)
			require.NoError(t, err)
			if tt.wantErr != nil {
				tt.assertion(t, err, fmt.Sprintf("%q. Simulator.SimulateOperation()", tt.name))
			} else {
				require.NoError(t, err)

			}
		})
	}
}
