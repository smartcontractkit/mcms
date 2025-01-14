package solana

import (
	"context"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

func TestTimelockInspector_IsOperation(t *testing.T) {
	t.Parallel()
	operationPDA, err := FindTimelockOperationPDA(testProgramID, testPDASeed, testOpID)
	require.NoError(t, err)

	operation := createTimelockOperation(t, 123)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    bool
		wantErr string
	}{
		{
			name: "success",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: true,
		},
		{
			name: "error: rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: not found",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, rpc.ErrNotFound)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.IsOperation(context.Background(), ContractAddress(testProgramID, testPDASeed), testOpID)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_IsOperationPending(t *testing.T) {
	t.Parallel()
	operationPDA, err := FindTimelockOperationPDA(testProgramID, testPDASeed, testOpID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    bool
		wantErr string
	}{
		{
			name: "operation is pending",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 123)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: true,
		},
		{
			name: "operation is done - not pending",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, TimelockOpDoneTimestamp)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: false,
		},
		{
			name: "invalid timestamp",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: false,
		},
		{
			name: "error: rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, TimelockOpDoneTimestamp)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.IsOperationPending(context.Background(), ContractAddress(testProgramID, testPDASeed), testOpID)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_IsOperationReady(t *testing.T) {
	t.Parallel()
	operationPDA, err := FindTimelockOperationPDA(testProgramID, testPDASeed, testOpID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    bool
		wantErr string
	}{
		{
			name: "operation is ready - timestamp same as block time",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 3)
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: true,
		},
		{
			name: "operation is ready - timestamp less than block time",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 2)
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: true,
		},
		{
			name: "operation is not ready - timestamp less than done",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 0)
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: false,
		},
		{
			name: "operation is not ready - operation is done",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 1) // timestamp is 1/done
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: false,
		},
		{
			name: "operation is not ready - timestamp more than block time",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 5) // timestamp > block time
				blockTime := solana.UnixTimeSeconds(1)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: false,
		},
		{
			name: "error: GetAccount rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 1)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: GetBlockHeight rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 1)
				blockTime := solana.UnixTimeSeconds(0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, errors.New("rpc error"), nil)
			},
			wantErr: "failed to get block height: rpc error",
		},
		{
			name: "error: GetBlockTime rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 1)
				blockTime := solana.UnixTimeSeconds(0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, errors.New("rpc error"))
			},
			wantErr: "failed to get block time: rpc error",
		},
		{
			name: "error: blocktime is nil",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 1)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, nil, nil, nil)
			},
			wantErr: "failed to get block time: nil value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.IsOperationReady(context.Background(), ContractAddress(testProgramID, testPDASeed), testOpID)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsOperationDone(t *testing.T) {
	t.Parallel()
	operationPDA, err := FindTimelockOperationPDA(testProgramID, testPDASeed, testOpID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    bool
		wantErr string
	}{
		{
			name: "operation is done",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, TimelockOpDoneTimestamp)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: true,
		},
		{
			name: "operation is not done",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: false,
		},
		{
			name: "error: rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, 0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.IsOperationDone(context.Background(), ContractAddress(testProgramID, testPDASeed), testOpID)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

// ----- helpers -----

func newTestTimelockInspector(t *testing.T) (*TimelockInspector, *mocks.JSONRPCClient) {
	t.Helper()
	jsonRPCClient := mocks.NewJSONRPCClient(t)
	inspector := NewTimelockInspector(rpc.NewWithCustomRPCClient(jsonRPCClient))

	return inspector, jsonRPCClient
}

func createTimelockOperation(t *testing.T, timestamp uint64) *timelock.Operation {
	t.Helper()

	pk, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	operation := &timelock.Operation{
		Timestamp:         timestamp,
		Id:                [32]uint8{},
		Predecessor:       [32]uint8{},
		Salt:              [32]uint8{},
		Authority:         pk.PublicKey(),
		IsFinalized:       false,
		TotalInstructions: 5,
		Instructions:      nil,
	}

	return operation
}
