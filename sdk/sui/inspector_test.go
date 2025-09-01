package sui

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mock_bindutils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mock_sui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewInspector(t *testing.T) {
	t.Parallel()
	mockClient := mock_sui.NewISuiAPI(t)
	mockSigner := mock_bindutils.NewSuiSigner(t)
	mcmsPackageId := "0x123456789abcdef"
	role := TimelockRoleProposer

	inspector, err := NewInspector(mockClient, mockSigner, mcmsPackageId, role)
	require.NoError(t, err)
	assert.NotNil(t, inspector)
	assert.Equal(t, mockClient, inspector.client)
	assert.Equal(t, mockSigner, inspector.signer)
	assert.Equal(t, mcmsPackageId, inspector.mcmsPackageId)
	assert.Equal(t, role, inspector.role)
	assert.NotNil(t, inspector.mcms)
}

func TestConfigTransformer_ToConfig(t *testing.T) {
	t.Parallel()

	transformer := NewConfigTransformer()

	t.Run("success - basic config transformation", func(t *testing.T) {
		t.Parallel()
		suiConfig := module_mcms.Config{
			Signers: []module_mcms.Signer{
				{
					Addr:  []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44},
					Index: 0,
					Group: 0,
				},
				{
					Addr:  []byte{0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
					Index: 1,
					Group: 1,
				},
			},
			GroupQuorums: []uint8{2, 1},
			GroupParents: []uint8{0, 0},
		}

		config, err := transformer.ToConfig(suiConfig)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify the transformation worked correctly
		assert.Equal(t, uint8(2), config.Quorum)
		assert.Len(t, config.Signers, 1)      // Group 0 signers
		assert.Len(t, config.GroupSigners, 1) // Group 1
	})

	t.Run("failure - empty config validation", func(t *testing.T) {
		t.Parallel()
		suiConfig := module_mcms.Config{
			Signers:      []module_mcms.Signer{},
			GroupQuorums: []uint8{},
			GroupParents: []uint8{},
		}

		config, err := transformer.ToConfig(suiConfig)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "Quorum must be greater than 0")
	})
}

func TestInspector_GetConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name      string
		mcmsAddr  string
		role      TimelockRole
		mockSetup func(m *mock_module_mcms.IMcms)
		want      *types.Config
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			role:     TimelockRoleProposer,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetConfig(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleProposer.Byte(),
				).Return(module_mcms.Config{
					Signers: []module_mcms.Signer{
						{
							Addr:  []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44},
							Index: 0,
							Group: 0,
						},
						{
							Addr:  []byte{0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
							Index: 1,
							Group: 1,
						},
					},
					GroupQuorums: []uint8{2, 1},
					GroupParents: []uint8{0, 0},
				}, nil)
			},
			want: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					common.BytesToAddress([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44}),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							common.BytesToAddress([]byte{0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name:     "failure - GetConfig failed",
			mcmsAddr: "0x123",
			role:     TimelockRoleBypasser,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetConfig(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleBypasser.Byte(),
				).Return(module_mcms.Config{}, errors.New("failed to get config"))
			},
			want:    nil,
			wantErr: AssertErrorContains("failed to GetConfig"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mock_sui.NewISuiAPI(t)
			mockSigner := mock_bindutils.NewSuiSigner(t)
			mockMcms := mock_module_mcms.NewIMcms(t)

			inspector := &Inspector{
				ConfigTransformer: ConfigTransformer{},
				client:            mockClient,
				signer:            mockSigner,
				mcmsPackageId:     "0x123456789abcdef",
				mcms:              mockMcms,
				role:              tt.role,
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMcms)
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
		role      TimelockRole
		mockSetup func(m *mock_module_mcms.IMcms)
		want      uint64
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			role:     TimelockRoleCanceller,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetOpCount(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleCanceller.Byte(),
				).Return(uint64(127), nil)
			},
			want:    127,
			wantErr: assert.NoError,
		},
		{
			name:     "failure - GetOpCount failed",
			mcmsAddr: "0x123",
			role:     TimelockRoleBypasser,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetOpCount(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleBypasser.Byte(),
				).Return(uint64(0), errors.New("failed to get op count"))
			},
			wantErr: AssertErrorContains("failed to GetOpCount"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mock_sui.NewISuiAPI(t)
			mockSigner := mock_bindutils.NewSuiSigner(t)
			mockMcms := mock_module_mcms.NewIMcms(t)

			inspector := &Inspector{
				ConfigTransformer: ConfigTransformer{},
				client:            mockClient,
				signer:            mockSigner,
				mcmsPackageId:     "0x123456789abcdef",
				mcms:              mockMcms,
				role:              tt.role,
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMcms)
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
		name      string
		mcmsAddr  string
		role      TimelockRole
		mockSetup func(m *mock_module_mcms.IMcms)
		wantRoot  common.Hash
		wantValid uint32
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			role:     TimelockRoleProposer,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				expectedRoot := common.HexToHash("0xabcdef1234567890")
				mockDevInspect.EXPECT().GetRoot(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleProposer.Byte(),
				).Return([]any{expectedRoot.Bytes(), uint64(12345)}, nil)
			},
			wantRoot:  common.HexToHash("0xabcdef1234567890"),
			wantValid: 12345,
			wantErr:   assert.NoError,
		},
		{
			name:     "failure - GetRoot failed",
			mcmsAddr: "0x123",
			role:     TimelockRoleCanceller,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetRoot(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleCanceller.Byte(),
				).Return(nil, errors.New("failed to get root"))
			},
			wantErr: AssertErrorContains("failed to GetRoot"),
		},
		{
			name:     "failure - invalid root result length",
			mcmsAddr: "0x123",
			role:     TimelockRoleProposer,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetRoot(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleProposer.Byte(),
				).Return([]any{common.Hash{}.Bytes()}, nil) // Only 1 element, should be 2
			},
			wantErr: AssertErrorContains("invalid root result: expected 2 elements, got 1"),
		},
		{
			name:     "failure - invalid root type",
			mcmsAddr: "0x123",
			role:     TimelockRoleProposer,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetRoot(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleProposer.Byte(),
				).Return([]any{"invalid_root_type", uint64(12345)}, nil)
			},
			wantErr: AssertErrorContains("invalid root type: expected []byte"),
		},
		{
			name:     "failure - invalid validUntil type",
			mcmsAddr: "0x123",
			role:     TimelockRoleProposer,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetRoot(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleProposer.Byte(),
				).Return([]any{common.Hash{}.Bytes(), "invalid_valid_until_type"}, nil)
			},
			wantErr: AssertErrorContains("invalid validUntil type: expected uint64"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mock_sui.NewISuiAPI(t)
			mockSigner := mock_bindutils.NewSuiSigner(t)
			mockMcms := mock_module_mcms.NewIMcms(t)

			inspector := &Inspector{
				ConfigTransformer: ConfigTransformer{},
				client:            mockClient,
				signer:            mockSigner,
				mcmsPackageId:     "0x123456789abcdef",
				mcms:              mockMcms,
				role:              tt.role,
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMcms)
			}

			gotRoot, gotValid, err := inspector.GetRoot(ctx, tt.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetRoot(%q)", tt.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.wantRoot, gotRoot)
			assert.Equal(t, tt.wantValid, gotValid)
		})
	}
}

func TestInspector_GetRootMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name      string
		mcmsAddr  string
		role      TimelockRole
		mockSetup func(m *mock_module_mcms.IMcms)
		want      types.ChainMetadata
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			mcmsAddr: "0x123",
			role:     TimelockRoleProposer,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetRootMetadata(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleProposer.Byte(),
				).Return(module_mcms.RootMetadata{
					PreOpCount: 42,
					Multisig:   "0xabcdef123456",
				}, nil)
			},
			want: types.ChainMetadata{
				StartingOpCount: 42,
				MCMAddress:      "0xabcdef123456",
			},
			wantErr: assert.NoError,
		},
		{
			name:     "failure - GetRootMetadata failed",
			mcmsAddr: "0x123",
			role:     TimelockRoleBypasser,
			mockSetup: func(m *mock_module_mcms.IMcms) {
				mockDevInspect := mock_module_mcms.NewIMcmsDevInspect(t)
				m.EXPECT().DevInspect().Return(mockDevInspect)
				mockDevInspect.EXPECT().GetRootMetadata(
					mock.Anything,
					mock.MatchedBy(func(opts *bind.CallOpts) bool {
						return opts.Signer != nil
					}),
					mock.MatchedBy(func(obj bind.Object) bool {
						return obj.Id == "0x123"
					}),
					TimelockRoleBypasser.Byte(),
				).Return(module_mcms.RootMetadata{}, errors.New("failed to get root metadata"))
			},
			want:    types.ChainMetadata{},
			wantErr: AssertErrorContains("failed to GetRootMetadata"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mock_sui.NewISuiAPI(t)
			mockSigner := mock_bindutils.NewSuiSigner(t)
			mockMcms := mock_module_mcms.NewIMcms(t)

			inspector := &Inspector{
				ConfigTransformer: ConfigTransformer{},
				client:            mockClient,
				signer:            mockSigner,
				mcmsPackageId:     "0x123456789abcdef",
				mcms:              mockMcms,
				role:              tt.role,
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mockMcms)
			}

			got, err := inspector.GetRootMetadata(ctx, tt.mcmsAddr)
			if !tt.wantErr(t, err, fmt.Sprintf("GetRootMetadata(%q)", tt.mcmsAddr)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
