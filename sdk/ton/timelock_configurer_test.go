package ton_test

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/smartcontractkit/chainlink-ton/cciplib/ton/tvm"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func TestTimelockConfigurer_UpdateDelay(t *testing.T) {
	t.Parallel()

	const validTimelockAddr = "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"

	tests := []struct {
		name            string
		timelockAddress string
		newDelay        uint64
		options         []mcmston.TimelockConfigurerOption
		mockSetup       func(m *ton_mocks.TonAPI)
		wantHash        string
		wantErr         string
		wantPrepared    bool
	}{
		{
			name:            "success",
			timelockAddress: validTimelockAddr,
			newDelay:        3600,
			mockSetup: func(m *ton_mocks.TonAPI) {
				m.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				apiw := ton_mocks.NewAPIClientWrapped(t)
				apiw.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
					Return(&tlb.Account{}, nil)
				apiw.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

				m.EXPECT().WaitForBlock(mock.Anything).Return(apiw)
				m.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{0xde, 0xad, 0xbe, 0xef}}, &ton.BlockIDExt{}, []byte{}, nil)
			},
			wantHash: "deadbeef",
		},
		{
			name:            "success - WithDoNotSendTimelockInstructionsOnChain option",
			timelockAddress: validTimelockAddr,
			newDelay:        3600,
			options: []mcmston.TimelockConfigurerOption{
				mcmston.WithDoNotSendTimelockInstructionsOnChain(),
			},
			mockSetup:    func(m *ton_mocks.TonAPI) {},
			wantPrepared: true,
		},
		{
			name:            "invalid timelock address",
			timelockAddress: "not-a-valid-ton-address",
			newDelay:        3600,
			mockSetup:       func(m *ton_mocks.TonAPI) {},
			wantErr:         "invalid timelock address",
		},
		{
			name:            "newDelay exceeds uint32 rejected",
			timelockAddress: validTimelockAddr,
			newDelay:        math.MaxUint32 + 1,
			mockSetup:       func(m *ton_mocks.TonAPI) {},
			wantErr:         "exceeds uint32 range",
		},
		{
			name:            "send transaction fails",
			timelockAddress: validTimelockAddr,
			newDelay:        3600,
			mockSetup: func(m *ton_mocks.TonAPI) {
				m.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				apiw := ton_mocks.NewAPIClientWrapped(t)
				apiw.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
					Return(&tlb.Account{}, nil)
				apiw.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

				m.EXPECT().WaitForBlock(mock.Anything).Return(apiw)
				m.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(nil, nil, nil, errors.New("boom"))
			},
			wantErr: "failed to send transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := ton_mocks.NewTonAPI(t)
			chainID := chaintest.Chain7TONID
			walletOperator := must(tvm.NewRandomV5R1TestWallet(api, chainID))

			tt.mockSetup(api)

			configurer := mcmston.NewTimelockConfigurer(walletOperator, tlb.MustFromTON("0.1"), tt.options...)
			result, err := configurer.UpdateDelay(t.Context(), tt.timelockAddress, tt.newDelay)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, result.Hash)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHash, result.Hash)
			if tt.wantPrepared {
				tx, ok := result.RawData.(types.Transaction)
				require.True(t, ok)
				assert.Equal(t, "RBACTimelock", tx.ContractType)
				assert.Equal(t, []string{"UpdateDelay"}, tx.Tags)
				body := must(cell.FromBOC(tx.Data))
				var msg timelock.UpdateDelay
				require.NoError(t, tlb.LoadFromCell(&msg, body.BeginParse()))

				result2, err := configurer.UpdateDelay(t.Context(), tt.timelockAddress, tt.newDelay)
				require.NoError(t, err)
				tx2, ok := result2.RawData.(types.Transaction)
				require.True(t, ok)
				body2 := must(cell.FromBOC(tx2.Data))
				var msg2 timelock.UpdateDelay
				require.NoError(t, tlb.LoadFromCell(&msg2, body2.BeginParse()))
				assert.Equal(t, tx.Data, tx2.Data)
				assert.Equal(t, msg.QueryID, msg2.QueryID)
				assert.NotZero(t, msg.QueryID)
			}
		})
	}
}

func TestTimelockConfigurer_GrantRole(t *testing.T) {
	t.Parallel()

	const validTimelockAddr = "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"
	validTargetAddr := address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8")

	tests := []struct {
		name            string
		timelockAddress string
		role            sdk.TimelockRole
		targetAddress   string
		options         []mcmston.TimelockConfigurerOption
		mockSetup       func(m *ton_mocks.TonAPI)
		wantHash        string
		wantErr         string
		wantPrepared    bool
	}{
		{
			name:            "success",
			timelockAddress: validTimelockAddr,
			role:            sdk.TimelockRoleProposer,
			targetAddress:   validTargetAddr.String(),
			mockSetup: func(m *ton_mocks.TonAPI) {
				m.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				apiw := ton_mocks.NewAPIClientWrapped(t)
				apiw.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
					Return(&tlb.Account{}, nil)
				apiw.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

				m.EXPECT().WaitForBlock(mock.Anything).Return(apiw)
				m.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{0xde, 0xad, 0xbe, 0xef}}, &ton.BlockIDExt{}, []byte{}, nil)
			},
			wantHash: "deadbeef",
		},
		{
			name:            "success - WithDoNotSendTimelockInstructionsOnChain option",
			timelockAddress: validTimelockAddr,
			role:            sdk.TimelockRoleProposer,
			targetAddress:   validTargetAddr.String(),
			options: []mcmston.TimelockConfigurerOption{
				mcmston.WithDoNotSendTimelockInstructionsOnChain(),
			},
			mockSetup:    func(m *ton_mocks.TonAPI) {},
			wantPrepared: true,
		},
		{
			name:            "success - admin role",
			timelockAddress: validTimelockAddr,
			role:            sdk.TimelockRoleAdmin,
			targetAddress:   validTargetAddr.String(),
			options: []mcmston.TimelockConfigurerOption{
				mcmston.WithDoNotSendTimelockInstructionsOnChain(),
			},
			mockSetup:    func(m *ton_mocks.TonAPI) {},
			wantPrepared: true,
		},
		{
			name:            "invalid timelock address",
			timelockAddress: "not-a-valid-ton-address",
			role:            sdk.TimelockRoleProposer,
			targetAddress:   validTargetAddr.String(),
			mockSetup:       func(m *ton_mocks.TonAPI) {},
			wantErr:         "invalid timelock address",
		},
		{
			name:            "invalid target address",
			timelockAddress: validTimelockAddr,
			role:            sdk.TimelockRoleProposer,
			targetAddress:   "not-a-valid-ton-address",
			mockSetup:       func(m *ton_mocks.TonAPI) {},
			wantErr:         "invalid target address",
		},
		{
			name:            "invalid timelock role",
			timelockAddress: validTimelockAddr,
			role:            sdk.TimelockRole(99),
			targetAddress:   validTargetAddr.String(),
			mockSetup:       func(m *ton_mocks.TonAPI) {},
			wantErr:         "invalid timelock role",
		},
		{
			name:            "send transaction fails",
			timelockAddress: validTimelockAddr,
			role:            sdk.TimelockRoleProposer,
			targetAddress:   validTargetAddr.String(),
			mockSetup: func(m *ton_mocks.TonAPI) {
				m.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				apiw := ton_mocks.NewAPIClientWrapped(t)
				apiw.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
					Return(&tlb.Account{}, nil)
				apiw.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

				m.EXPECT().WaitForBlock(mock.Anything).Return(apiw)
				m.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(nil, nil, nil, errors.New("boom"))
			},
			wantErr: "failed to send transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := ton_mocks.NewTonAPI(t)
			chainID := chaintest.Chain7TONID
			walletOperator := must(tvm.NewRandomV5R1TestWallet(api, chainID))

			tt.mockSetup(api)

			configurer := mcmston.NewTimelockConfigurer(walletOperator, tlb.MustFromTON("0.1"), tt.options...)
			result, err := configurer.GrantRole(t.Context(), tt.timelockAddress, tt.role, tt.targetAddress)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, result.Hash)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHash, result.Hash)
			if tt.wantPrepared {
				tx, ok := result.RawData.(types.Transaction)
				require.True(t, ok)
				assert.Equal(t, "RBACTimelock", tx.ContractType)
				assert.Equal(t, []string{"RBACTimelock", "GrantRole"}, tx.Tags)
				body := must(cell.FromBOC(tx.Data))
				var msg rbac.GrantRole
				require.NoError(t, tlb.LoadFromCell(&msg, body.BeginParse()))

				roleHash, err := mcmston.TimelockRoleHash(tt.role)
				require.NoError(t, err)
				assert.Equal(t, roleHash, msg.Role.Value())
				assert.Equal(t, address.MustParseAddr(tt.targetAddress), msg.Account)
				assert.NotZero(t, msg.QueryID)

				result2, err := configurer.GrantRole(t.Context(), tt.timelockAddress, tt.role, tt.targetAddress)
				require.NoError(t, err)
				tx2, ok := result2.RawData.(types.Transaction)
				require.True(t, ok)
				body2 := must(cell.FromBOC(tx2.Data))
				var msg2 rbac.GrantRole
				require.NoError(t, tlb.LoadFromCell(&msg2, body2.BeginParse()))
				assert.Equal(t, tx.Data, tx2.Data)
				assert.Equal(t, msg.QueryID, msg2.QueryID)
			}
		})
	}
}
