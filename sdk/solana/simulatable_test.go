package solana

import (
	"context"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

func TestSimulatable_SimulationOn(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pk, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	pubKey := pk.PublicKey()

	someError := errors.New("some error")
	tests := []struct {
		name          string
		expectedError string
		actionToRun   func(client *rpc.Client, s *Simulatable) error
		setupMock     func(t *testing.T, client *mocks.JSONRPCClient)
	}{
		{
			name: "Successful Simulation",
			setupMock: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, client, 50, nil, nil)
			},
			actionToRun: func(client *rpc.Client, s *Simulatable) error {
				instruction := mcm.NewClearSignersInstruction(testPDASeed, pubKey, pubKey, pubKey)
				_, err = s.sendAndConfirmOrSimulate(ctx, client, pk, instruction, rpc.CommitmentFinalized)

				return err
			},
			expectedError: "",
		},
		{
			name: "Simulated Action Error",
			actionToRun: func(_ *rpc.Client, _ *Simulatable) error {
				return someError
			},
			expectedError: "some error",
		},
		{
			name: "Instruction Validation Error",
			actionToRun: func(client *rpc.Client, s *Simulatable) error {
				instruction := mcm.NewClearSignersInstruction(testPDASeed, pubKey, pubKey, pubKey)
				// setting MultisigId to nil will trigger validation error later when building the instruction
				instruction.MultisigId = nil
				_, err = s.sendAndConfirmOrSimulate(ctx, client, pk, instruction, rpc.CommitmentConfirmed)

				return err
			},
			expectedError: "simulation failed: unable to validate and build instruction: MultisigId parameter is not set",
		},
		{
			name: "SimulateTransaction RPC Communication Error",
			setupMock: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, client, 50, someError, nil)
			},
			actionToRun: func(client *rpc.Client, s *Simulatable) error {
				instruction := mcm.NewClearSignersInstruction(testPDASeed, pubKey, pubKey, pubKey)
				_, err = s.sendAndConfirmOrSimulate(ctx, client, pk, instruction, rpc.CommitmentConfirmed)

				return err
			},
			expectedError: "some error",
		},
		{
			name: "SimulateTransaction RPC Response Error",
			setupMock: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, client, 50, nil, someError)
			},
			actionToRun: func(client *rpc.Client, s *Simulatable) error {
				instruction := mcm.NewClearSignersInstruction(testPDASeed, pubKey, pubKey, pubKey)
				_, err = s.sendAndConfirmOrSimulate(ctx, client, pk, instruction, rpc.CommitmentConfirmed)

				return err
			},
			expectedError: "some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &Simulatable{}

			mockJSONRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)

			if tt.setupMock != nil {
				tt.setupMock(t, mockJSONRPCClient)
			}

			action := func() error {
				return tt.actionToRun(client, s)
			}

			err = s.EnableSimulation(ctx, client, pk, rpc.SimulateTransactionOpts{}, action)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSimulatable_SimulationOff(t *testing.T) {
	t.Parallel()

	s := &Simulatable{}
	mockJSONRPCClient := mocks.NewJSONRPCClient(t)
	client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)
	ctx := context.Background()

	pk, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	pubKey := pk.PublicKey()

	// expect actual transaction to be sent
	mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
		"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya", nil)

	instruction := mcm.NewClearSignersInstruction(testPDASeed, pubKey, pubKey, pubKey)

	_, err = s.sendAndConfirmOrSimulate(ctx, client, pk, instruction, rpc.CommitmentConfirmed)

	require.NoError(t, err)
}
