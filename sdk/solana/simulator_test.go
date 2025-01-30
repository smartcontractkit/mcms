package solana

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestSimulator_SimulateSetRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	metadata := types.ChainMetadata{StartingOpCount: 100, MCMAddress: ContractAddress(testMCMProgramID, testPDASeed)}
	proof := []common.Hash{common.HexToHash("0x1"), common.HexToHash("0x2")}
	root := common.HexToHash("0x1234")
	validUntil := uint32(2082758400)
	signatures := []types.Signature{
		{R: common.HexToHash("0x3"), S: common.HexToHash("0x4"), V: 27},
		{R: common.HexToHash("0x5"), S: common.HexToHash("0x6"), V: 27},
	}

	tests := []struct {
		name          string
		setupMocks    func(t *testing.T, client *mocks.JSONRPCClient)
		expectedError string
	}{
		{
			name: "success",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, client, 50, nil, nil)
			},
		},
		{
			name: "failure: GetLastBlockhash error",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, client, 50, errors.New("GetLastBlockhash error"), nil)
			},
			expectedError: "GetLastBlockhash error",
		},
		{
			name: "failure: SimulateTransaction error",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, client, 50, nil, errors.New("SimulateTransaction error"))
			},
			expectedError: "SimulateTransaction error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor, client := newTestExecutor(t, auth, testChainSelector)
			simulator := NewSimulator(executor)

			tt.setupMocks(t, client)

			err = simulator.SimulateSetRoot(ctx, "", metadata, proof, root, validUntil, signatures)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
