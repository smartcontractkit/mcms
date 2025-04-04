package aptos

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	mock_aptossdk "github.com/smartcontractkit/mcms/sdk/aptos/mocks/aptos"
	mock_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewConfigurer(t *testing.T) {
	t.Parallel()
	mockClient := mock_aptossdk.NewAptosRpcClient(t)
	mockSigner := mock_aptossdk.NewTransactionSigner(t)

	configurer := NewConfigurer(mockClient, mockSigner, TimelockRoleBypasser)
	assert.NotNil(t, configurer)
	assert.Equal(t, mockClient, configurer.client)
	assert.Equal(t, mockSigner, configurer.auth)
	assert.Equal(t, TimelockRoleBypasser, configurer.role)
	assert.NotNil(t, configurer.bindingFn)
}

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	type args struct {
		mcmsAddress string
		cfg         *types.Config
		clearRoot   bool
	}
	tests := []struct {
		name      string
		args      args
		role      TimelockRole
		mockSetup func(m *mock_mcms.MCMS)
		want      types.TransactionResult
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				mcmsAddress: "0x123",
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
			role: TimelockRoleCanceller,
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().SetConfig(
					mock.Anything,
					TimelockRoleCanceller.Byte(),
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
					// Quorums - group 0 has 2, group 1 has 1
					[]uint8{2, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					// Group Parents -
					//   group 0 => group 0
					//   group 1 => group 0
					//   group 2 => group 1
					[]uint8{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					true,
				).Return(&api.PendingTransaction{
					Hash: "0x123456789",
				}, nil)
			},
			want: types.TransactionResult{
				Hash:        "0x123456789",
				ChainFamily: cselectors.FamilyAptos,
				RawData: &api.PendingTransaction{
					Hash: "0x123456789",
				},
			},
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				mcmsAddress: "invalidaddress!",
			},
			wantErr: AssertErrorContains("parse"),
		}, {
			name: "failure - SetConfig failed",
			args: args{
				mcmsAddress: "0x1234",
				cfg: &types.Config{
					Quorum:  1,
					Signers: []common.Address{common.HexToAddress("0x1")},
				},
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().SetConfig(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("error during SetConfig"))
			},
			wantErr: AssertErrorContains("error during SetConfig"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			configurer := Configurer{
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
				role: tt.role,
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := configurer.SetConfig(ctx, tt.args.mcmsAddress, tt.args.cfg, tt.args.clearRoot)
			if !tt.wantErr(t, err, fmt.Sprintf("SetConfig(%v, %v, %v)", tt.args.mcmsAddress, tt.args.cfg, tt.args.clearRoot)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
