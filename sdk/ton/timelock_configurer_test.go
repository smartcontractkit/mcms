package ton_test

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"

	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
)

func TestTimelockConfigurer_UpdateDelay(t *testing.T) {
	t.Parallel()

	const validTimelockAddr = "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"

	tests := []struct {
		name            string
		timelockAddress string
		newDelay        uint64
		mockSetup       func(m *ton_mocks.TonAPI)
		wantHash        string
		wantErr         string
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

			configurer := mcmston.NewTimelockConfigurer(walletOperator, tlb.MustFromTON("0.1"))
			result, err := configurer.UpdateDelay(t.Context(), tt.timelockAddress, tt.newDelay)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, result.Hash)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHash, result.Hash)
		})
	}
}
