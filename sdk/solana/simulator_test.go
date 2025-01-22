package solana

import (
	"context"
	"errors"
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
	// test with nil client
	simulator, err = NewSimulator(nil, auth, encoder)
	require.EqualError(t, err, "Simulator was created without a Solana RPC client")
	require.Nil(t, simulator)

	// test with nil encoder
	simulator, err = NewSimulator(client, auth, nil)
	require.EqualError(t, err, "Simulator was created without an encoder")
	require.Nil(t, simulator)
}

func TestSimulator_SimulateOperation(t *testing.T) {
	t.Parallel()

	type args struct {
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
	require.NoError(t, err)
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
		{
			name: "error: invalid additional fields",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: contractID,
				},

				op: types.Operation{
					Transaction: types.Transaction{
						AdditionalFields: []byte("invalid"),
					},
					ChainSelector: types.ChainSelector(selector),
				},
			},
			mockSetup: func(m *mocks.JSONRPCClient) {},
			wantErr:   fmt.Errorf("unable to unmarshal additional fields: invalid character 'i' looking for beginning of value"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to unmarshal additional fields: invalid character 'i' looking for beginning of value")
			},
		},
		{
			name: "error: block hash fetch failed",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: contractID,
				},

				op: types.Operation{
					Transaction:   tx,
					ChainSelector: types.ChainSelector(selector),
				},
			},
			mockSetup: func(m *mocks.JSONRPCClient) {
				mockSolanaSimulation(t, m, 20, 5, "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", errors.New("failed to simulate"))
			},
			wantErr: errors.New("failed to simulate"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to get recent blockhash: failed to simulate")
			},
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
			if tt.wantErr != nil {
				tt.assertion(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
