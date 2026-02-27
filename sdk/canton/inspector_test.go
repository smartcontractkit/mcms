package canton

import (
	"context"
	"io"
	"testing"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	mock_apiv2 "github.com/smartcontractkit/mcms/sdk/canton/mocks/apiv2"
	"github.com/smartcontractkit/mcms/types"
)

// mockGetActiveContractsClient implements the streaming client for GetActiveContracts
type mockGetActiveContractsClient struct {
	grpc.ClientStream
	responses []*apiv2.GetActiveContractsResponse
	index     int
}

func (m *mockGetActiveContractsClient) Recv() (*apiv2.GetActiveContractsResponse, error) {
	if m.index >= len(m.responses) {
		return nil, io.EOF
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *mockGetActiveContractsClient) CloseSend() error {
	return nil
}

func TestNewInspector(t *testing.T) {
	t.Parallel()

	mockStateClient := mock_apiv2.NewStateServiceClient(t)
	party := "Alice::party123"
	role := TimelockRoleProposer

	inspector := NewInspector(mockStateClient, party, role)

	require.NotNil(t, inspector)
	assert.Equal(t, mockStateClient, inspector.stateClient)
	assert.Equal(t, party, inspector.party)
	assert.Equal(t, role, inspector.role)
	assert.Nil(t, inspector.contractCache)
}

func TestInspector_GetConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mcmsAddr  string
		role      TimelockRole
		mockSetup func(*mock_apiv2.StateServiceClient) *mcms.MCMS
		want      *types.Config
		wantErr   string
	}{
		{
			name:     "success - proposer role",
			mcmsAddr: "contract-id-123",
			role:     TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) *mcms.MCMS {
				setupMockGetMCMSContract(t, mockClient, "contract-id-123", &mcms.MCMS{
					Owner:      "Alice::party123",
					InstanceId: "instance-123",
					ChainId:    1,
					Proposer: mcms.RoleState{
						Config: mcms.MultisigConfig{
							Signers: []mcms.SignerInfo{
								{
									SignerAddress: "1122334455667788",
									SignerGroup:   0,
									SignerIndex:   0,
								},
								{
									SignerAddress: "2233445566778899",
									SignerGroup:   1,
									SignerIndex:   1,
								},
							},
							GroupQuorums: []cantontypes.INT64{2, 1},
							GroupParents: []cantontypes.INT64{0, 0},
						},
					},
				})
				return nil
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("0x1122334455667788"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							common.HexToAddress("0x2233445566778899"),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			wantErr: "",
		},
		{
			name:     "success - bypasser role",
			mcmsAddr: "contract-id-456",
			role:     TimelockRoleBypasser,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) *mcms.MCMS {
				setupMockGetMCMSContract(t, mockClient, "contract-id-456", &mcms.MCMS{
					Owner:      "Bob::party456",
					InstanceId: "instance-456",
					ChainId:    2,
					Bypasser: mcms.RoleState{
						Config: mcms.MultisigConfig{
							Signers: []mcms.SignerInfo{
								{
									SignerAddress: "aabbccddeeff0011",
									SignerGroup:   0,
									SignerIndex:   0,
								},
							},
							GroupQuorums: []cantontypes.INT64{1},
							GroupParents: []cantontypes.INT64{0},
						},
					},
				})
				return nil
			},
			want: &types.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("0xaabbccddeeff0011"),
				},
				GroupSigners: []types.Config{},
			},
			wantErr: "",
		},
		{
			name:     "success - canceller role",
			mcmsAddr: "contract-id-789",
			role:     TimelockRoleCanceller,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) *mcms.MCMS {
				setupMockGetMCMSContract(t, mockClient, "contract-id-789", &mcms.MCMS{
					Owner:      "Carol::party789",
					InstanceId: "instance-789",
					ChainId:    3,
					Canceller: mcms.RoleState{
						Config: mcms.MultisigConfig{
							Signers: []mcms.SignerInfo{
								{
									SignerAddress: "ffeeddccbbaa9988",
									SignerGroup:   0,
									SignerIndex:   0,
								},
							},
							GroupQuorums: []cantontypes.INT64{1},
							GroupParents: []cantontypes.INT64{0},
						},
					},
				})
				return nil
			},
			want: &types.Config{
				Quorum: 1,
				Signers: []common.Address{
					common.HexToAddress("0xffeeddccbbaa9988"),
				},
				GroupSigners: []types.Config{},
			},
			wantErr: "",
		},
		{
			name:     "failure - contract not found",
			mcmsAddr: "nonexistent-contract",
			role:     TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) *mcms.MCMS {
				mockClient.EXPECT().GetLedgerEnd(mock.Anything, mock.Anything).Return(
					&apiv2.GetLedgerEndResponse{
						Offset: 123,
					},
					nil,
				)

				streamClient := &mockGetActiveContractsClient{
					responses: []*apiv2.GetActiveContractsResponse{},
				}
				mockClient.EXPECT().GetActiveContracts(mock.Anything, mock.Anything).Return(
					streamClient,
					nil,
				)
				return nil
			},
			want:    nil,
			wantErr: "MCMS contract with ID nonexistent-contract not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockStateClient := mock_apiv2.NewStateServiceClient(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mockStateClient)
			}

			inspector := NewInspector(mockStateClient, "Alice::party123", tt.role)

			got, err := inspector.GetConfig(ctx, tt.mcmsAddr)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestInspector_GetOpCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mcmsAddr  string
		role      TimelockRole
		mockSetup func(*mock_apiv2.StateServiceClient)
		want      uint64
		wantErr   string
	}{
		{
			name:     "success - proposer role",
			mcmsAddr: "contract-id-123",
			role:     TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-123", &mcms.MCMS{
					Proposer: mcms.RoleState{
						ExpiringRoot: mcms.ExpiringRoot{
							OpCount: 5,
						},
					},
				})
			},
			want:    5,
			wantErr: "",
		},
		{
			name:     "success - bypasser role",
			mcmsAddr: "contract-id-456",
			role:     TimelockRoleBypasser,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-456", &mcms.MCMS{
					Bypasser: mcms.RoleState{
						ExpiringRoot: mcms.ExpiringRoot{
							OpCount: 10,
						},
					},
				})
			},
			want:    10,
			wantErr: "",
		},
		{
			name:     "success - canceller role with zero op count",
			mcmsAddr: "contract-id-789",
			role:     TimelockRoleCanceller,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-789", &mcms.MCMS{
					Canceller: mcms.RoleState{
						ExpiringRoot: mcms.ExpiringRoot{
							OpCount: 0,
						},
					},
				})
			},
			want:    0,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockStateClient := mock_apiv2.NewStateServiceClient(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mockStateClient)
			}

			inspector := NewInspector(mockStateClient, "Alice::party123", tt.role)

			got, err := inspector.GetOpCount(ctx, tt.mcmsAddr)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestInspector_GetRoot(t *testing.T) {
	t.Parallel()

	validUntilTime := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name           string
		mcmsAddr       string
		role           TimelockRole
		mockSetup      func(*mock_apiv2.StateServiceClient)
		wantRoot       common.Hash
		wantValidUntil uint32
		wantErr        string
	}{
		{
			name:     "success - proposer role",
			mcmsAddr: "contract-id-123",
			role:     TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-123", &mcms.MCMS{
					Proposer: mcms.RoleState{
						ExpiringRoot: mcms.ExpiringRoot{
							Root:       "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
							ValidUntil: cantontypes.TIMESTAMP(validUntilTime),
						},
					},
				})
			},
			wantRoot:       common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
			wantValidUntil: uint32(validUntilTime.Unix()),
			wantErr:        "",
		},
		{
			name:     "success - bypasser role",
			mcmsAddr: "contract-id-456",
			role:     TimelockRoleBypasser,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-456", &mcms.MCMS{
					Bypasser: mcms.RoleState{
						ExpiringRoot: mcms.ExpiringRoot{
							Root:       "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
							ValidUntil: cantontypes.TIMESTAMP(validUntilTime),
						},
					},
				})
			},
			wantRoot:       common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
			wantValidUntil: uint32(validUntilTime.Unix()),
			wantErr:        "",
		},
		{
			name:     "failure - invalid root hex",
			mcmsAddr: "contract-id-bad",
			role:     TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-bad", &mcms.MCMS{
					Proposer: mcms.RoleState{
						ExpiringRoot: mcms.ExpiringRoot{
							Root:       "invalid-hex-string",
							ValidUntil: cantontypes.TIMESTAMP(validUntilTime),
						},
					},
				})
			},
			wantRoot:       common.Hash{},
			wantValidUntil: 0,
			wantErr:        "failed to decode root hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockStateClient := mock_apiv2.NewStateServiceClient(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mockStateClient)
			}

			inspector := NewInspector(mockStateClient, "Alice::party123", tt.role)

			gotRoot, gotValidUntil, err := inspector.GetRoot(ctx, tt.mcmsAddr)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRoot, gotRoot)
				assert.Equal(t, tt.wantValidUntil, gotValidUntil)
			}
		})
	}
}

func TestInspector_GetRootMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mcmsAddr  string
		role      TimelockRole
		mockSetup func(*mock_apiv2.StateServiceClient)
		want      types.ChainMetadata
		wantErr   string
	}{
		{
			name:     "success - proposer role",
			mcmsAddr: "contract-id-123",
			role:     TimelockRoleProposer,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-123", &mcms.MCMS{
					InstanceId: "instance-123",
					Proposer: mcms.RoleState{
						RootMetadata: mcms.RootMetadata{
							ChainId:     1,
							PreOpCount:  5,
							PostOpCount: 10,
						},
					},
				})
			},
			want: types.ChainMetadata{
				StartingOpCount: 5,
				MCMAddress:      "instance-123",
			},
			wantErr: "",
		},
		{
			name:     "success - bypasser role",
			mcmsAddr: "contract-id-456",
			role:     TimelockRoleBypasser,
			mockSetup: func(mockClient *mock_apiv2.StateServiceClient) {
				setupMockGetMCMSContract(t, mockClient, "contract-id-456", &mcms.MCMS{
					InstanceId: "instance-456",
					Bypasser: mcms.RoleState{
						RootMetadata: mcms.RootMetadata{
							ChainId:     2,
							PreOpCount:  0,
							PostOpCount: 3,
						},
					},
				})
			},
			want: types.ChainMetadata{
				StartingOpCount: 0,
				MCMAddress:      "instance-456",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockStateClient := mock_apiv2.NewStateServiceClient(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mockStateClient)
			}

			inspector := NewInspector(mockStateClient, "Alice::party123", tt.role)

			got, err := inspector.GetRootMetadata(ctx, tt.mcmsAddr)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestToConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		bindConfig mcms.MultisigConfig
		want       *types.Config
		wantErr    string
	}{
		{
			name: "success - simple single group",
			bindConfig: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{
						SignerAddress: "1122334455667788",
						SignerGroup:   0,
						SignerIndex:   0,
					},
					{
						SignerAddress: "2233445566778899",
						SignerGroup:   0,
						SignerIndex:   1,
					},
				},
				GroupQuorums: []cantontypes.INT64{2},
				GroupParents: []cantontypes.INT64{0},
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("0x1122334455667788"),
					common.HexToAddress("0x2233445566778899"),
				},
				GroupSigners: []types.Config{},
			},
			wantErr: "",
		},
		{
			name: "success - hierarchical groups",
			bindConfig: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{
						SignerAddress: "1122334455667788",
						SignerGroup:   0,
						SignerIndex:   0,
					},
					{
						SignerAddress: "2233445566778899",
						SignerGroup:   1,
						SignerIndex:   1,
					},
					{
						SignerAddress: "3344556677889900",
						SignerGroup:   1,
						SignerIndex:   2,
					},
				},
				GroupQuorums: []cantontypes.INT64{2, 2},
				GroupParents: []cantontypes.INT64{0, 0},
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("0x1122334455667788"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("0x2233445566778899"),
							common.HexToAddress("0x3344556677889900"),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "failure - empty config",
			bindConfig: mcms.MultisigConfig{
				Signers:      []mcms.SignerInfo{},
				GroupQuorums: []cantontypes.INT64{},
				GroupParents: []cantontypes.INT64{},
			},
			want:    nil,
			wantErr: "Quorum must be greater than 0",
		},
		{
			name: "failure - group index exceeds maximum",
			bindConfig: mcms.MultisigConfig{
				Signers: []mcms.SignerInfo{
					{
						SignerAddress: "1122334455667788",
						SignerGroup:   32, // Exceeds maximum of 31
						SignerIndex:   0,
					},
				},
				GroupQuorums: []cantontypes.INT64{1},
				GroupParents: []cantontypes.INT64{0},
			},
			want:    nil,
			wantErr: "signer group index 32 exceeds maximum of 31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := toConfig(tt.bindConfig)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// Helper function to setup mock for getMCMSContract
func setupMockGetMCMSContract(t *testing.T, mockClient *mock_apiv2.StateServiceClient, contractID string, mcmsContract *mcms.MCMS) {
	// Mock GetLedgerEnd
	mockClient.EXPECT().GetLedgerEnd(mock.Anything, mock.Anything).Return(
		&apiv2.GetLedgerEndResponse{
			Offset: 123,
		},
		nil,
	)

	// Create the created event for the MCMS contract
	createdEvent := &apiv2.CreatedEvent{
		ContractId: contractID,
		TemplateId: &apiv2.Identifier{
			PackageId:  "#mcms",
			ModuleName: "MCMS.Main",
			EntityName: "MCMS",
		},
	}

	// Create response with the active contract
	responses := []*apiv2.GetActiveContractsResponse{
		{
			ContractEntry: &apiv2.GetActiveContractsResponse_ActiveContract{
				ActiveContract: &apiv2.ActiveContract{
					CreatedEvent: createdEvent,
				},
			},
		},
	}

	streamClient := &mockGetActiveContractsClient{
		responses: responses,
	}

	mockClient.EXPECT().GetActiveContracts(mock.Anything, mock.Anything).Return(
		streamClient,
		nil,
	)
}
