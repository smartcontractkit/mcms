package ton_test

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"
	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
)

// TestConfigurer_SetConfig tests the SetConfig method of the Configurer.
func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	signers := testutils.MakeNewECDSASigners(16)

	tests := []struct {
		name      string
		mcmAddr   string
		cfg       *types.Config
		clearRoot bool
		mockSetup func(m *ton_mocks.TonAPI)
		want      string
		wantErr   error
	}{
		{
			name:    "success",
			mcmAddr: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			cfg: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[1].Address(),
					signers[2].Address(),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							signers[3].Address(),
						},
						GroupSigners: nil,
					},
				},
			},
			clearRoot: true,
			mockSetup: func(m *ton_mocks.TonAPI) {
				// Mock CurrentMasterchainInfo
				m.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				// Mock WaitForBlock
				apiw := ton_mocks.NewAPIClientWrapped(t)
				apiw.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
					Return(&tlb.Account{}, nil)

				apiw.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

				m.EXPECT().WaitForBlock(mock.Anything).
					Return(apiw)

				m.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, nil)
			},
			want:    "010203040e",
			wantErr: nil,
		},
		{
			name:    "failure - SendTransaction fails",
			mcmAddr: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			cfg: &types.Config{
				Quorum: 2,
				Signers: []common.Address{
					signers[1].Address(),
					signers[2].Address(),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							signers[3].Address(),
						},
						GroupSigners: nil,
					},
				},
			},
			clearRoot: false,
			mockSetup: func(m *ton_mocks.TonAPI) {
				// Mock CurrentMasterchainInfo
				m.EXPECT().CurrentMasterchainInfo(mock.Anything).
					Return(&ton.BlockIDExt{}, nil)

				// Mock WaitForBlock
				apiw := ton_mocks.NewAPIClientWrapped(t)
				apiw.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
					Return(&tlb.Account{}, nil)

				apiw.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

				m.EXPECT().WaitForBlock(mock.Anything).
					Return(apiw)

				// Mock SendTransaction to return an error
				m.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, errors.New("transaction failed"))
			},
			want:    "",
			wantErr: errors.New("transaction failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_api := ton_mocks.NewTonAPI(t)
			chainID := chaintest.Chain7TONID
			walletOperator := must(tvm.NewRandomV5R1TestWallet(_api, chainID))

			// Apply the mock setup for the ContractDeployBackend
			if tt.mockSetup != nil {
				tt.mockSetup(_api)
			}

			// Create the Configurer instance
			configurer, err := tonmcms.NewConfigurer(walletOperator, tlb.MustFromTON("0.1"))
			require.NoError(t, err)

			// Call SetConfig
			tx, err := configurer.SetConfig(ctx, tt.mcmAddr, tt.cfg, tt.clearRoot)

			// Assert the results
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				assert.Empty(t, tx.Hash)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, tx.Hash)
			}
		})
	}
}
