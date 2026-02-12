package canton

import (
	"context"
	"encoding/json"
	"testing"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mock_apiv2 "github.com/smartcontractkit/mcms/sdk/canton/mocks/apiv2"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	encoder := NewEncoder(1, 5, false)
	mockStateClient := mock_apiv2.NewStateServiceClient(t)
	inspector := NewInspector(mockStateClient, "Alice::party123", TimelockRoleProposer)
	mockCommandClient := mock_apiv2.NewCommandServiceClient(t)
	userId := "user123"
	party := "Alice::party123"
	role := TimelockRoleProposer

	executor, err := NewExecutor(encoder, inspector, mockCommandClient, userId, party, role)

	require.NoError(t, err)
	require.NotNil(t, executor)
	assert.Equal(t, encoder, executor.Encoder)
	assert.Equal(t, inspector, executor.Inspector)
	assert.Equal(t, mockCommandClient, executor.client)
	assert.Equal(t, userId, executor.userId)
	assert.Equal(t, party, executor.party)
	assert.Equal(t, role, executor.role)
}

func TestExecutor_ExecuteOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		metadata  types.ChainMetadata
		nonce     uint32
		proof     []common.Hash
		op        types.Operation
		mockSetup func(*mock_apiv2.CommandServiceClient)
		wantErr   string
	}{
		{
			name: "success - execute operation",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-123",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig"
				}`),
			},
			nonce: 1,
			proof: []common.Hash{
				common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
				common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
			},
			op: types.Operation{
				Transaction: types.Transaction{
					To:   "target-contract",
					Data: []byte{0x11, 0x22, 0x33},
					AdditionalFields: json.RawMessage(`{
						"targetInstanceId": "instance-123",
						"functionName": "executeAction",
						"operationData": "112233",
						"targetCid": "cid-123",
						"contractIds": ["cid-456", "cid-789"]
					}`),
				},
			},
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(
					mock.Anything,
					mock.MatchedBy(func(req *apiv2.SubmitAndWaitForTransactionRequest) bool {
						return req.Commands != nil &&
							req.Commands.WorkflowId == "mcms-execute-op" &&
							len(req.Commands.Commands) == 1
					}),
				).Return(&apiv2.SubmitAndWaitForTransactionResponse{
					Transaction: &apiv2.Transaction{
						UpdateId: "tx-execute-123",
						Events: []*apiv2.Event{
							{
								Event: &apiv2.Event_Created{
									Created: &apiv2.CreatedEvent{
										ContractId: "new-contract-id-after-execute",
										TemplateId: &apiv2.Identifier{
											PackageId:  "mcms-package",
											ModuleName: "MCMS.Main",
											EntityName: "MCMS",
										},
									},
								},
							},
						},
					},
				}, nil)
			},
			wantErr: "",
		},
		{
			name: "failure - missing targetInstanceId",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-123",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig"
				}`),
			},
			nonce: 1,
			proof: []common.Hash{},
			op: types.Operation{
				Transaction: types.Transaction{
					To:   "target-contract",
					Data: []byte{0x11, 0x22, 0x33},
					AdditionalFields: json.RawMessage(`{
						"functionName": "executeAction",
						"operationData": "112233",
						"targetCid": "cid-123"
					}`),
				},
			},
			mockSetup: nil,
			wantErr:   "targetInstanceId is required",
		},
		{
			name: "failure - missing functionName",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-123",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig"
				}`),
			},
			nonce: 1,
			proof: []common.Hash{},
			op: types.Operation{
				Transaction: types.Transaction{
					To:   "target-contract",
					Data: []byte{0x11, 0x22, 0x33},
					AdditionalFields: json.RawMessage(`{
						"targetInstanceId": "instance-123",
						"operationData": "112233",
						"targetCid": "cid-123"
					}`),
				},
			},
			mockSetup: nil,
			wantErr:   "functionName is required",
		},
		{
			name: "failure - missing targetCid",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-123",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig"
				}`),
			},
			nonce: 1,
			proof: []common.Hash{},
			op: types.Operation{
				Transaction: types.Transaction{
					To:   "target-contract",
					Data: []byte{0x11, 0x22, 0x33},
					AdditionalFields: json.RawMessage(`{
						"targetInstanceId": "instance-123",
						"functionName": "executeAction",
						"operationData": "112233"
					}`),
				},
			},
			mockSetup: nil,
			wantErr:   "targetCid is required",
		},
		{
			name: "failure - submission error",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-bad",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig"
				}`),
			},
			nonce: 1,
			proof: []common.Hash{},
			op: types.Operation{
				Transaction: types.Transaction{
					To:   "target-contract",
					Data: []byte{0x11, 0x22, 0x33},
					AdditionalFields: json.RawMessage(`{
						"targetInstanceId": "instance-123",
						"functionName": "executeAction",
						"operationData": "112233",
						"targetCid": "cid-123"
					}`),
				},
			},
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(mock.Anything, mock.Anything).Return(
					nil,
					assert.AnError,
				)
			},
			wantErr: "failed to execute operation",
		},
		{
			name: "failure - no MCMS created event after execution",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-no-event",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig"
				}`),
			},
			nonce: 1,
			proof: []common.Hash{},
			op: types.Operation{
				Transaction: types.Transaction{
					To:   "target-contract",
					Data: []byte{0x11, 0x22, 0x33},
					AdditionalFields: json.RawMessage(`{
						"targetInstanceId": "instance-123",
						"functionName": "executeAction",
						"operationData": "112233",
						"targetCid": "cid-123"
					}`),
				},
			},
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(mock.Anything, mock.Anything).Return(
					&apiv2.SubmitAndWaitForTransactionResponse{
						Transaction: &apiv2.Transaction{
							UpdateId: "tx-no-event",
							Events:   []*apiv2.Event{}, // No events
						},
					},
					nil,
				)
			},
			wantErr: "execute-op tx had no Created MCMS event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			encoder := NewEncoder(1, 5, false)
			mockStateClient := mock_apiv2.NewStateServiceClient(t)
			inspector := NewInspector(mockStateClient, "Alice::party123", TimelockRoleProposer)
			mockCommandClient := mock_apiv2.NewCommandServiceClient(t)

			if tt.mockSetup != nil {
				tt.mockSetup(mockCommandClient)
			}

			executor, err := NewExecutor(encoder, inspector, mockCommandClient, "user123", "Alice::party123", TimelockRoleProposer)
			require.NoError(t, err)

			result, err := executor.ExecuteOperation(ctx, tt.metadata, tt.nonce, tt.proof, tt.op)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.Hash)
				assert.NotNil(t, result.RawData)
				assert.Contains(t, result.RawData, "NewMCMSContractID")
			}
		})
	}
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		metadata         types.ChainMetadata
		proof            []common.Hash
		root             [32]byte
		validUntil       uint32
		sortedSignatures []types.Signature
		mockSetup        func(*mock_apiv2.CommandServiceClient)
		wantErr          string
	}{
		{
			name: "success - set root",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-123",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig",
					"preOpCount": 0,
					"postOpCount": 5
				}`),
			},
			proof: []common.Hash{
				common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
			},
			root:       [32]byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef},
			validUntil: 1234567890,
			sortedSignatures: []types.Signature{
				{
					V: 27,
					R: common.Hash{0x11},
					S: common.Hash{0x22},
				},
			},
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(
					mock.Anything,
					mock.MatchedBy(func(req *apiv2.SubmitAndWaitForTransactionRequest) bool {
						return req.Commands != nil &&
							req.Commands.WorkflowId == "mcms-set-root" &&
							len(req.Commands.Commands) == 1
					}),
				).Return(&apiv2.SubmitAndWaitForTransactionResponse{
					Transaction: &apiv2.Transaction{
						UpdateId: "tx-setroot-123",
						Events: []*apiv2.Event{
							{
								Event: &apiv2.Event_Created{
									Created: &apiv2.CreatedEvent{
										ContractId: "new-contract-id-after-setroot",
										TemplateId: &apiv2.Identifier{
											PackageId:  "mcms-package",
											ModuleName: "MCMS.Main",
											EntityName: "MCMS",
										},
									},
								},
							},
						},
					},
				}, nil)
			},
			wantErr: "",
		},
		{
			name: "failure - submission error",
			metadata: types.ChainMetadata{
				MCMAddress:      "contract-id-bad",
				StartingOpCount: 0,
				AdditionalFields: json.RawMessage(`{
					"chainId": 1,
					"multisigId": "test-multisig",
					"preOpCount": 0,
					"postOpCount": 5
				}`),
			},
			proof:            []common.Hash{},
			root:             [32]byte{0x12},
			validUntil:       1234567890,
			sortedSignatures: []types.Signature{},
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(mock.Anything, mock.Anything).Return(
					nil,
					assert.AnError,
				)
			},
			wantErr: "failed to set root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			encoder := NewEncoder(1, 5, false)
			mockStateClient := mock_apiv2.NewStateServiceClient(t)
			inspector := NewInspector(mockStateClient, "Alice::party123", TimelockRoleProposer)
			mockCommandClient := mock_apiv2.NewCommandServiceClient(t)

			if tt.mockSetup != nil {
				tt.mockSetup(mockCommandClient)
			}

			executor, err := NewExecutor(encoder, inspector, mockCommandClient, "user123", "Alice::party123", TimelockRoleProposer)
			require.NoError(t, err)

			result, err := executor.SetRoot(ctx, tt.metadata, tt.proof, tt.root, tt.validUntil, tt.sortedSignatures)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.Hash)
				assert.NotNil(t, result.RawData)
			}
		})
	}
}
