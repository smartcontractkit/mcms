package solana

import (
	"context"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/access_controller"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

func TestTimelockInspector_GetProposers(t *testing.T) {
	t.Parallel()

	timelockConfigPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)

	config := createTimelockConfig(t)
	controller := createAccessController(t)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    []string
		wantErr string
	}{
		{
			name: "success",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.ProposerRoleAccessController, controller, nil)
			},
			want: []string{controller.AccessList.Xs[0].String(), controller.AccessList.Xs[1].String()},
		},
		{
			name: "error: get timelock config account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: get controller account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.ProposerRoleAccessController, controller, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.GetProposers(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_GetExecutors(t *testing.T) {
	t.Parallel()

	timelockConfigPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)

	config := createTimelockConfig(t)
	controller := createAccessController(t)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    []string
		wantErr string
	}{
		{
			name: "success",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.ExecutorRoleAccessController, controller, nil)
			},
			want: []string{controller.AccessList.Xs[0].String(), controller.AccessList.Xs[1].String()},
		},
		{
			name: "error: get timelock config account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: get controller account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.ExecutorRoleAccessController, controller, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.GetExecutors(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_GetBypassers(t *testing.T) {
	t.Parallel()

	timelockConfigPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)

	config := createTimelockConfig(t)
	controller := createAccessController(t)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    []string
		wantErr string
	}{
		{
			name: "success",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.BypasserRoleAccessController, controller, nil)
			},
			want: []string{controller.AccessList.Xs[0].String(), controller.AccessList.Xs[1].String()},
		},
		{
			name: "error: get timelock config account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: get controller account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.BypasserRoleAccessController, controller, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.GetBypassers(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_GetCancellers(t *testing.T) {
	t.Parallel()

	timelockConfigPDA, err := FindTimelockConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)

	config := createTimelockConfig(t)
	controller := createAccessController(t)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    []string
		wantErr string
	}{
		{
			name: "success",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.CancellerRoleAccessController, controller, nil)
			},
			want: []string{controller.AccessList.Xs[0].String(), controller.AccessList.Xs[1].String()},
		},
		{
			name: "error: get timelock config account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: get controller account info rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockGetAccountInfo(t, mockJSONRPCClient, timelockConfigPDA, config, nil)
				mockGetAccountInfo(t, mockJSONRPCClient, config.CancellerRoleAccessController, controller, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inspector, jsonRPCClient := newTestTimelockInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.GetCancellers(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed))
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_IsOperation(t *testing.T) {
	t.Parallel()
	operationPDA, err := FindTimelockOperationPDA(testTimelockProgramID, testPDASeed, testOpID)
	require.NoError(t, err)

	operation := createTimelockOperation(t, timelock.Scheduled_OperationState)

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

			got, err := inspector.IsOperation(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed), testOpID)
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
	operationPDA, err := FindTimelockOperationPDA(testTimelockProgramID, testPDASeed, testOpID)
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
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: true,
		},
		{
			name: "operation is done - not pending",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Done_OperationState)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: false,
		},
		{
			name: "error: rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
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

			got, err := inspector.IsOperationPending(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed), testOpID)
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
	operationPDA, err := FindTimelockOperationPDA(testTimelockProgramID, testPDASeed, testOpID)
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
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				operation.Timestamp = 3
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: true,
		},
		{
			name: "operation is ready - timestamp less than block time",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				operation.Timestamp = 2
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: true,
		},
		{
			name: "operation is not ready - operation is done",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Done_OperationState)
				blockTime := solana.UnixTimeSeconds(3)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: false,
		},
		{
			name: "operation is not ready - timestamp more than block time",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Done_OperationState)
				operation.Timestamp = 5
				blockTime := solana.UnixTimeSeconds(1)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, nil)
			},
			want: false,
		},
		{
			name: "error: GetAccount rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, errors.New("rpc error"))
			},
			wantErr: "rpc error",
		},
		{
			name: "error: GetBlockHeight rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				blockTime := solana.UnixTimeSeconds(0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, errors.New("rpc error"), nil)
			},
			wantErr: "failed to get block height: rpc error",
		},
		{
			name: "error: GetBlockTime rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				blockTime := solana.UnixTimeSeconds(0)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
				mockGetBlockTime(t, mockJSONRPCClient, 1, &blockTime, nil, errors.New("rpc error"))
			},
			wantErr: "failed to get block time: rpc error",
		},
		{
			name: "error: blocktime is nil",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
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

			got, err := inspector.IsOperationReady(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed), testOpID)
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
	operationPDA, err := FindTimelockOperationPDA(testTimelockProgramID, testPDASeed, testOpID)
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
				operation := createTimelockOperation(t, timelock.Done_OperationState)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: true,
		},
		{
			name: "operation is not done",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Scheduled_OperationState)
				mockGetAccountInfo(t, mockJSONRPCClient, operationPDA, operation, nil)
			},
			want: false,
		},
		{
			name: "error: rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				operation := createTimelockOperation(t, timelock.Done_OperationState)
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

			got, err := inspector.IsOperationDone(context.Background(), ContractAddress(testTimelockProgramID, testPDASeed), testOpID)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestTimelockInspector_getRoleAccessController(t *testing.T) {
	t.Parallel()

	config := createTimelockConfig(t)

	tests := []struct {
		name    string
		role    timelock.Role
		want    solana.PublicKey
		wantErr string
	}{
		{
			name: "success: get proposer access controller",
			role: timelock.Proposer_Role,
			want: config.ProposerRoleAccessController,
		},
		{
			name: "success: get executor access controller",

			role: timelock.Executor_Role,
			want: config.ExecutorRoleAccessController,
		},
		{
			name: "success: get canceller access controller",
			role: timelock.Canceller_Role,
			want: config.CancellerRoleAccessController,
		},
		{
			name:    "error: admin role not supported",
			role:    timelock.Admin_Role,
			wantErr: "not supported role: Admin",
		},
		{
			name:    "error: unknown role",
			role:    timelock.Role(100),
			wantErr: "unknown role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getRoleAccessController(*config, tt.role)
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

func createTimelockOperation(t *testing.T, state timelock.OperationState) *timelock.Operation {
	t.Helper()

	operation := &timelock.Operation{
		Timestamp:         0,
		Id:                [32]uint8{},
		Predecessor:       [32]uint8{},
		Salt:              [32]uint8{},
		TotalInstructions: 5,
		Instructions:      nil,
		State:             state,
	}

	return operation
}

func createTimelockConfig(t *testing.T) *timelock.Config {
	t.Helper()

	proposerKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	executorKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	cancellerKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	bypasserKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	config := &timelock.Config{
		TimelockId:                    [32]uint8{},
		Owner:                         solana.PublicKey{},
		ProposedOwner:                 solana.PublicKey{},
		ProposerRoleAccessController:  proposerKey.PublicKey(),
		ExecutorRoleAccessController:  executorKey.PublicKey(),
		CancellerRoleAccessController: cancellerKey.PublicKey(),
		BypasserRoleAccessController:  bypasserKey.PublicKey(),
		MinDelay:                      0,
		BlockedSelectors:              timelock.BlockedSelectors{},
	}

	return config
}

func createAccessController(t *testing.T) *access_controller.AccessController {
	t.Helper()

	pk, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	pk2, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	accessList := access_controller.AccessList{
		Len: 2,
		Xs:  [64]solana.PublicKey{pk.PublicKey(), pk2.PublicKey()},
	}

	ac := &access_controller.AccessController{
		AccessList: accessList,
	}

	return ac
}
