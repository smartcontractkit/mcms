package solana

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"

	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/mcm"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestSimulatorSimulateSetRoot(t *testing.T) {
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
	previousRoot := common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdef")
	rootAndOpCount := &bindings.ExpiringRootAndOpCount{Root: previousRoot, ValidUntil: 123}
	opCountPDA, err := FindExpiringRootAndOpCountPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)

	tests := []struct {
		name          string
		setupMocks    func(t *testing.T, client *mocks.JSONRPCClient)
		expectedError string
	}{
		{
			name: "success",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, client, opCountPDA, rootAndOpCount, nil)
				mockSolanaSimulateTransaction(t, client, 50, nil, nil)
			},
		},
		{
			name: "failure: unable to GetRoot",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, client, opCountPDA, root, errors.New("error"))
			},
			expectedError: "failed to get root: error",
		},
		{
			name: "failure: GetRoot returns the same root",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				currentRootAndOpCount := &bindings.ExpiringRootAndOpCount{Root: root, ValidUntil: 123}
				mockGetAccountInfo(t, client, opCountPDA, currentRootAndOpCount, nil)
			},
			expectedError: "SignedHashAlreadySeen: 0x0000000000000000000000000000000000000000000000000000000000001234",
		},
		{
			name: "failure: GetLastBlockhash error",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, client, opCountPDA, rootAndOpCount, nil)
				mockSolanaSimulateTransaction(t, client, 50, errors.New("GetLastBlockhash error"), nil)
			},
			expectedError: "GetLastBlockhash error",
		},
		{
			name: "failure: SimulateTransaction error",
			setupMocks: func(t *testing.T, client *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, client, opCountPDA, rootAndOpCount, nil)
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

func TestSimulatorSimulateOperation(t *testing.T) {
	t.Parallel()

	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	selector := cselectors.SOLANA_DEVNET.Selector

	testWallet, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	ix, err := system.NewTransferInstruction(20*solana.LAMPORTS_PER_SOL, auth.PublicKey(), testWallet.PublicKey()).ValidateAndBuild()
	require.NoError(t, err)

	data, err := ix.Data()
	require.NoError(t, err)

	tx, err := NewTransaction(solana.SystemProgramID.String(), data, nil, ix.Accounts(), "solana-testing", []string{})
	require.NoError(t, err)

	tests := []struct {
		name          string
		givenOP       types.Operation
		setupMocks    func(t *testing.T, client *mocks.JSONRPCClient)
		expectedError string
	}{
		{
			name: "success: SimulateOperation",
			givenOP: types.Operation{
				Transaction:   tx,
				ChainSelector: types.ChainSelector(selector),
			},
			setupMocks: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, m, 50, nil, nil)
			},
		},
		{
			name: "error: invalid additional fields",
			givenOP: types.Operation{
				Transaction: types.Transaction{
					AdditionalFields: []byte("invalid"),
				},
				ChainSelector: types.ChainSelector(selector),
			},
			setupMocks: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
			},
			expectedError: "unable to unmarshal additional fields: invalid character 'i' looking for beginning of value",
		},
		{
			name: "error: block hash fetch failed",
			givenOP: types.Operation{
				Transaction:   tx,
				ChainSelector: types.ChainSelector(selector),
			},
			setupMocks: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaSimulateTransaction(t, m, 50, errors.New("block hash error"), nil)
			},
			expectedError: "unable to simulate instruction: block hash error",
		},
		{
			name: "failure: SimulateTransaction error",
			givenOP: types.Operation{
				Transaction:   tx,
				ChainSelector: types.ChainSelector(selector),
			},
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

			err = simulator.SimulateOperation(context.Background(), types.ChainMetadata{
				MCMAddress: fmt.Sprintf("%s.%s", testMCMProgramID.String(), testPDASeed),
			}, tt.givenOP)

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
