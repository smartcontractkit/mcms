package aptos

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	module_mcms "github.com/smartcontractkit/chainlink-aptos/bindings/mcms/mcms"

	mock_aptossdk "github.com/smartcontractkit/mcms/sdk/aptos/mocks/aptos"
	mock_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewInspector(t *testing.T) {
	t.Parallel()
	mockClient := mock_aptossdk.NewAptosRpcClient(t)

	inspector := NewInspector(mockClient)
	assert.NotNil(t, inspector)
	assert.Equal(t, mockClient, inspector.client)
}

func TestInspector_GetConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name      string
		mcmsAddr  string
		mockSetup func(m *mock_mcms.MCMS)
		want      *types.Config
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetConfig(mock.Anything).Return(module_mcms.Config{
					Signers: []module_mcms.Signer{
						{
							Addr:  common.HexToAddress("0x111").Bytes(),
							Index: 0,
							Group: 0,
						}, {
							Addr:  common.HexToAddress("0x222").Bytes(),
							Index: 1,
							Group: 1,
						}, {
							Addr:  common.HexToAddress("0x333").Bytes(),
							Index: 2,
							Group: 0,
						}, {
							Addr:  common.HexToAddress("0x444").Bytes(),
							Index: 3,
							Group: 2,
						}, {
							Addr:  common.HexToAddress("0x555").Bytes(),
							Index: 4,
							Group: 2,
						},
					},
					GroupQuorums: []byte{2, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					GroupParents: []byte{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				}, nil)
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.HexToAddress("0x111"),
					common.HexToAddress("0x333"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							common.HexToAddress("0x222"),
						},
						GroupSigners: []types.Config{
							{
								Quorum: 2,
								Signers: []common.Address{
									common.HexToAddress("0x444"),
									common.HexToAddress("0x555"),
								},
								GroupSigners: []types.Config{},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		}, {
			name:     "failure - invalid MCMS address",
			mcmsAddr: "invalidaddress",
			wantErr:  AssertErrorContains("parse MCMS address"),
		}, {
			name:     "failure - GetConfig failed",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetConfig(mock.Anything).Return(module_mcms.Config{}, errors.New("error during GetConfig"))
			},
			want:    nil,
			wantErr: AssertErrorContains("error during GetConfig"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := Inspector{
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.GetConfig(ctx, tt.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetConfig(%q)", tt.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInspector_GetOpCount(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name      string
		mcmsAddr  string
		mockSetup func(m *mock_mcms.MCMS)
		want      uint64
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetOpCount(mock.Anything).Return(127, nil)
			},
			want:    127,
			wantErr: assert.NoError,
		}, {
			name:     "failure - invalid MCMS address",
			mcmsAddr: "invalidaddress",
			wantErr:  AssertErrorContains("parse MCMS address"),
		}, {
			name:     "failure - GetOpCount failed",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetOpCount(mock.Anything).Return(0, errors.New("error during GetOpCount"))
			},
			wantErr: AssertErrorContains("error during GetOpCount"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := Inspector{
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.GetOpCount(ctx, tt.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetOpCount(%q)", tt.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInspector_GetRoot(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name           string
		mcmsAddr       string
		mockSetup      func(m *mock_mcms.MCMS)
		wantHash       common.Hash
		wantValidUntil uint32
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetRoot(mock.Anything).Return(common.HexToAddress("0x123456789").Bytes(), 1742933811, nil)
			},
			wantHash:       common.HexToHash("0x123456789"),
			wantValidUntil: 1742933811,
			wantErr:        assert.NoError,
		}, {
			name:     "failure - invalid MCMS address",
			mcmsAddr: "invalidaddress",
			wantErr:  AssertErrorContains("parse MCMS address"),
		}, {
			name:     "failure - GetRoot failed",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetRoot(mock.Anything).Return(nil, 0, errors.New("error during GetRoot"))
			},
			wantErr: AssertErrorContains("error during GetRoot"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := Inspector{
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			gotHash, gotValidUntil, err := inspector.GetRoot(ctx, tt.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetRoot(%q)", tt.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.wantHash, gotHash)
			assert.Equal(t, tt.wantValidUntil, gotValidUntil)
		})
	}
}

func TestInspector_GetRootMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name      string
		mcmsAddr  string
		mockSetup func(m *mock_mcms.MCMS)
		want      types.ChainMetadata
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetRootMetadata(mock.Anything).Return(module_mcms.RootMetadata{
					Multisig:   aptos.AccountThree,
					PreOpCount: 201,
				}, nil)
			},
			want: types.ChainMetadata{
				StartingOpCount: 201,
				MCMAddress:      aptos.AccountThree.StringLong(),
			},
			wantErr: assert.NoError,
		}, {
			name:     "failure - invalid MCMS address",
			mcmsAddr: "invalidaddress",
			wantErr:  AssertErrorContains("parse MCMS address"),
		}, {
			name:     "failure - GetRootMetadata failed",
			mcmsAddr: "0x123",
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().GetRootMetadata(mock.Anything).Return(module_mcms.RootMetadata{}, errors.New("error during GetRootMetadata"))
			},
			wantErr: AssertErrorContains("error during GetRootMetadata"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			inspector := Inspector{
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := inspector.GetRootMetadata(ctx, tt.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetRootMetadata(%q)", tt.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
