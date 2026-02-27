package canton

import (
	"context"
	"testing"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mock_apiv2 "github.com/smartcontractkit/mcms/sdk/canton/mocks/apiv2"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewConfigurer(t *testing.T) {
	t.Parallel()

	mockCommandClient := mock_apiv2.NewCommandServiceClient(t)
	userId := "user123"
	party := "Alice::party123"
	role := TimelockRoleProposer

	configurer, err := NewConfigurer(mockCommandClient, userId, party, role)

	require.NoError(t, err)
	require.NotNil(t, configurer)
	assert.Equal(t, mockCommandClient, configurer.client)
	assert.Equal(t, userId, configurer.userId)
	assert.Equal(t, party, configurer.party)
	assert.Equal(t, role, configurer.role)
}

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mcmsAddr   string
		cfg        *types.Config
		clearRoot  bool
		role       TimelockRole
		mockSetup  func(*mock_apiv2.CommandServiceClient)
		wantErr    string
		wantTxHash string
	}{
		{
			name:     "success - simple config",
			mcmsAddr: "contract-id-123",
			cfg: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("0x1111111111111111111111111111111111111111"),
					common.HexToAddress("0x2222222222222222222222222222222222222222"),
				},
				GroupSigners: []types.Config{},
			},
			clearRoot: false,
			role:      TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				// Mock successful submission
				mockClient.EXPECT().SubmitAndWaitForTransaction(
					mock.Anything,
					mock.MatchedBy(func(req *apiv2.SubmitAndWaitForTransactionRequest) bool {
						return req.Commands != nil &&
							req.Commands.WorkflowId == "mcms-set-config" &&
							len(req.Commands.Commands) == 1 &&
							req.Commands.Commands[0].GetExercise() != nil
					}),
				).Return(&apiv2.SubmitAndWaitForTransactionResponse{
					Transaction: &apiv2.Transaction{
						UpdateId: "tx-123",
						Events: []*apiv2.Event{
							{
								Event: &apiv2.Event_Created{
									Created: &apiv2.CreatedEvent{
										ContractId: "new-contract-id-456",
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
			wantTxHash: "tx.Digest",
			wantErr:    "",
		},
		{
			name:     "success - hierarchical config",
			mcmsAddr: "contract-id-456",
			cfg: &types.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("0x1111111111111111111111111111111111111111"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("0x2222222222222222222222222222222222222222"),
							common.HexToAddress("0x3333333333333333333333333333333333333333"),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			clearRoot: true,
			role:      TimelockRoleBypasser,
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(mock.Anything, mock.Anything).Return(
					&apiv2.SubmitAndWaitForTransactionResponse{
						Transaction: &apiv2.Transaction{
							UpdateId: "tx-456",
							Events: []*apiv2.Event{
								{
									Event: &apiv2.Event_Created{
										Created: &apiv2.CreatedEvent{
											ContractId: "new-contract-id-789",
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
					},
					nil,
				)
			},
			wantTxHash: "tx.Digest",
			wantErr:    "",
		},
		{
			name:     "failure - submission error",
			mcmsAddr: "contract-id-bad",
			cfg: &types.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("0x1111111111111111111111111111111111111111"),
				},
				GroupSigners: []types.Config{},
			},
			clearRoot: false,
			role:      TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.CommandServiceClient) {
				mockClient.EXPECT().SubmitAndWaitForTransaction(mock.Anything, mock.Anything).Return(
					nil,
					assert.AnError,
				)
			},
			wantTxHash: "",
			wantErr:    "failed to set config",
		},
		{
			name:     "failure - no MCMS created event",
			mcmsAddr: "contract-id-no-event",
			cfg: &types.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("0x1111111111111111111111111111111111111111"),
				},
				GroupSigners: []types.Config{},
			},
			clearRoot: false,
			role:      TimelockRoleProposer,
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
			wantTxHash: "",
			wantErr:    "set-config tx had no Created MCMS event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockCommandClient := mock_apiv2.NewCommandServiceClient(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mockCommandClient)
			}

			configurer, err := NewConfigurer(mockCommandClient, "user123", "Alice::party123", tt.role)
			require.NoError(t, err)

			result, err := configurer.SetConfig(ctx, tt.mcmsAddr, tt.cfg, tt.clearRoot)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTxHash, result.Hash)
				assert.NotNil(t, result.RawData)
				assert.Contains(t, result.RawData, "NewMCMSContractID")
				assert.Contains(t, result.RawData, "NewMCMSTemplateID")
			}
		})
	}
}
