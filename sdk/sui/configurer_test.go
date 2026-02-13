package sui

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-sui/bindings/bind"

	mockbindutils "github.com/smartcontractkit/mcms/sdk/sui/mocks/bindutils"
	mockmodulemcms "github.com/smartcontractkit/mcms/sdk/sui/mocks/mcms"
	mocksui "github.com/smartcontractkit/mcms/sdk/sui/mocks/sui"
	"github.com/smartcontractkit/mcms/types"
)

func AssertErrorContains(errorMessage string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, msgAndArgs ...any) bool {
		if err == nil {
			return assert.Fail(t, "Expected error to be returned", msgAndArgs...)
		}

		return assert.Contains(t, err.Error(), errorMessage, msgAndArgs...)
	}
}

func MustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)

	return data
}
func TestNewConfigurer(t *testing.T) {
	t.Parallel()
	mockClient := mocksui.NewISuiAPI(t)
	mockSigner := mockbindutils.NewSuiSigner(t)

	// Test successful creation
	mcmsPackageID := "0x123456789abcdef"
	ownerCap := "0xabcdef123456789"
	chainSelector := chainsel.SUI_TESTNET.Selector

	configurer, err := NewConfigurer(mockClient, mockSigner, TimelockRoleBypasser, mcmsPackageID, ownerCap, chainSelector)
	require.NoError(t, err)
	assert.NotNil(t, configurer)
	assert.Equal(t, mockClient, configurer.client)
	assert.Equal(t, mockSigner, configurer.signer)
	assert.Equal(t, TimelockRoleBypasser, configurer.role)
	assert.Equal(t, ownerCap, configurer.ownerCap)
	assert.Equal(t, chainSelector, configurer.chainSelector)
	assert.NotNil(t, configurer.mcms)
}

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	type args struct {
		mcmsAddr  string
		cfg       *types.Config
		clearRoot bool
	}

	// Create test transaction response
	expectedTx := &models.SuiTransactionBlockResponse{
		Digest: "0x123456789abcdef",
	}

	tests := []struct {
		name         string
		args         args
		role         TimelockRole
		chainID      uint64
		mockSetup    func(mockmcms *mockmodulemcms.IMcms)
		want         types.TransactionResult
		wantErr      assert.ErrorAssertionFunc
		wantChainErr bool
	}{
		{
			name: "success with bypasser role",
			args: args{
				mcmsAddr: "0x123",
				cfg: &types.Config{
					Quorum: 2,
					Signers: []common.Address{
						common.HexToAddress("0x333"),
						common.HexToAddress("0x111"),
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
								},
							},
						},
					},
				},
				clearRoot: true,
			},
			role:    TimelockRoleBypasser,
			chainID: chainsel.SUI_TESTNET.Selector,
			mockSetup: func(mockmcms *mockmodulemcms.IMcms) {
				expectedChainID := new(big.Int).SetUint64(chainsel.SUI_TESTNET.ChainID)
				mockmcms.EXPECT().SetConfig(
					mock.Anything,
					mock.Anything,
					bind.Object{Id: "0xownerCap"},
					bind.Object{Id: "0x123"},
					TimelockRoleBypasser.Byte(),
					expectedChainID,
					// Ordered addresses
					[][]byte{
						common.HexToAddress("0x111").Bytes(),
						common.HexToAddress("0x222").Bytes(),
						common.HexToAddress("0x333").Bytes(),
						common.HexToAddress("0x444").Bytes(),
						common.HexToAddress("0x555").Bytes(),
					},
					// Groups - 0x111 & 0x333 are in group 0, 0x222 is in group 1, 0x444 & 0x555 is in group 2
					[]uint8{0, 1, 0, 2, 2},
					// Quorums - group 0 has 2, group 1 has 1, group 2 has 2
					[]uint8{2, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					// Group Parents -
					//   group 0 => group 0
					//   group 1 => group 0
					//   group 2 => group 1
					[]uint8{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					true,
				).Return(expectedTx, nil)
			},
			want: types.TransactionResult{
				Hash:        "0x123456789abcdef",
				ChainFamily: chainsel.FamilySui,
				RawData:     expectedTx,
			},
			wantErr: assert.NoError,
		},
		{
			name: "success with canceller role",
			args: args{
				mcmsAddr: "0x456",
				cfg: &types.Config{
					Quorum:  1,
					Signers: []common.Address{common.HexToAddress("0x1")},
				},
				clearRoot: false,
			},
			role:    TimelockRoleCanceller,
			chainID: chainsel.SUI_TESTNET.Selector,
			mockSetup: func(mockmcms *mockmodulemcms.IMcms) {
				expectedChainID := new(big.Int).SetUint64(chainsel.SUI_TESTNET.ChainID)
				mockmcms.EXPECT().SetConfig(
					mock.Anything,
					mock.Anything,
					bind.Object{Id: "0xownerCap"},
					bind.Object{Id: "0x456"},
					TimelockRoleCanceller.Byte(),
					expectedChainID,
					[][]byte{common.HexToAddress("0x1").Bytes()},
					[]uint8{0},
					// First quorum is 1, rest are 0
					[]uint8{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					// All group parents are 0
					[]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					false,
				).Return(expectedTx, nil)
			},
			want: types.TransactionResult{
				Hash:        "0x123456789abcdef",
				ChainFamily: chainsel.FamilySui,
				RawData:     expectedTx,
			},
			wantErr: assert.NoError,
		},
		{
			name: "failure - invalid chain selector",
			args: args{
				mcmsAddr: "0x789",
				cfg: &types.Config{
					Quorum:  1,
					Signers: []common.Address{common.HexToAddress("0x1")},
				},
			},
			role:         TimelockRoleProposer,
			chainID:      999999, // Invalid chain ID that doesn't map to Sui
			wantChainErr: true,
			wantErr:      AssertErrorContains("chain id not found"),
		},
		{
			name: "failure - SetConfig transaction failed",
			args: args{
				mcmsAddr: "0x123",
				cfg: &types.Config{
					Quorum:  1,
					Signers: []common.Address{common.HexToAddress("0x1")},
				},
			},
			role:    TimelockRoleProposer,
			chainID: chainsel.SUI_TESTNET.Selector,
			mockSetup: func(mockmcms *mockmodulemcms.IMcms) {
				mockmcms.EXPECT().SetConfig(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("transaction failed"))
			},
			wantErr: AssertErrorContains("failed to set config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mocksui.NewISuiAPI(t)
			mockSigner := mockbindutils.NewSuiSigner(t)
			Modulemockmcms := mockmodulemcms.NewIMcms(t)

			configurer := &Configurer{
				client:        mockClient,
				signer:        mockSigner,
				role:          tt.role,
				mcms:          Modulemockmcms,
				ownerCap:      "0xownerCap",
				chainSelector: tt.chainID,
			}

			if tt.mockSetup != nil {
				tt.mockSetup(Modulemockmcms)
			}

			got, err := configurer.SetConfig(ctx, tt.args.mcmsAddr, tt.args.cfg, tt.args.clearRoot)
			if !tt.wantErr(t, err, fmt.Sprintf("SetConfig(%v, %v, %v)", tt.args.mcmsAddr, tt.args.cfg, tt.args.clearRoot)) {
				return
			}
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
