package ton_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"

	mcmston "github.com/smartcontractkit/mcms/sdk/ton"
	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
)

func TestTimelockExecutor_NewTimelockExecutor(t *testing.T) {
	t.Parallel()

	chainID := chaintest.Chain7TONID
	amount := tlb.MustFromTON("0.1")

	tests := []struct {
		name    string
		mutate  func(opts mcmston.TimelockExecutorOpts) mcmston.TimelockExecutorOpts
		wantErr string
	}{
		{
			name: "success",
			mutate: func(opts mcmston.TimelockExecutorOpts) mcmston.TimelockExecutorOpts {
				return opts
			},
			wantErr: "",
		},
		{
			name: "nil client",
			mutate: func(opts mcmston.TimelockExecutorOpts) mcmston.TimelockExecutorOpts {
				opts.Client = nil

				return opts
			},
			wantErr: "failed to create sdk.Executor - client (ton.APIClientWrapped) is nil",
		},
		{
			name: "nil wallet",
			mutate: func(opts mcmston.TimelockExecutorOpts) mcmston.TimelockExecutorOpts {
				opts.Wallet = nil

				return opts
			},
			wantErr: "failed to create sdk.Executor - wallet (*wallet.Wallet) is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_api := ton_mocks.NewTonAPI(t)
			walletOperator := must(tvm.NewRandomV5R1TestWallet(_api, chainID))
			var client ton.APIClientWrapped = ton_mocks.NewAPIClientWrapped(t)

			opts := tt.mutate(mcmston.TimelockExecutorOpts{
				Client: client,
				Wallet: walletOperator,
				Amount: amount,
			})

			executor, err := mcmston.NewTimelockExecutor(opts)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, executor)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, executor)
		})
	}
}

func TestTimelockExecutor_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sharedMockSetup := func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
		// Mock CurrentMasterchainInfo
		api.EXPECT().CurrentMasterchainInfo(mock.Anything).
			Return(&ton.BlockIDExt{}, nil)

		// Mock WaitForBlock
		client.EXPECT().GetAccount(mock.Anything, mock.Anything, mock.Anything).
			Return(&tlb.Account{}, nil)

		client.EXPECT().RunGetMethod(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(ton.NewExecutionResult([]any{big.NewInt(5)}), nil)

		api.EXPECT().WaitForBlock(mock.Anything).
			Return(client)
	}

	tests := []struct {
		name            string
		timelockAddress string
		bop             types.BatchOperation
		predecessor     common.Hash
		salt            common.Hash
		mockSetup       func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped)
		wantTxHash      string
		wantErr         error
	}{
		{
			name:            "success",
			timelockAddress: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			bop: types.BatchOperation{
				ChainSelector: chaintest.Chain7Selector,
				Transactions: []types.Transaction{
					{
						To:               "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
						Data:             cell.BeginCell().MustStoreBinarySnake([]byte{1, 2, 3}).EndCell().ToBOC(),
						AdditionalFields: json.RawMessage(`{"value": 0}`)},
				},
			},
			mockSetup: func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
				// Successful tx send
				sharedMockSetup(api, client)

				// Mock SendTransaction to return (no error)
				api.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, nil)
			},
			wantTxHash: "010203040e",
			wantErr:    nil,
		},
		{
			name:            "failure in tx execution",
			timelockAddress: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			bop: types.BatchOperation{
				ChainSelector: chaintest.Chain7Selector,
				Transactions: []types.Transaction{
					{
						To:               "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
						Data:             cell.BeginCell().MustStoreBinarySnake([]byte{1, 2, 3}).EndCell().ToBOC(),
						AdditionalFields: json.RawMessage(`{"value": 0}`)},
				},
			},
			mockSetup: func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
				// Error tx send
				sharedMockSetup(api, client)

				// Mock SendTransaction to return an error
				api.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, errors.New("error during tx send"))
			},
			wantTxHash: "",
			wantErr:    errors.New("failed to send transaction: error during tx send"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Initialize the mock
			chainID := chaintest.Chain7TONID
			_api := ton_mocks.NewTonAPI(t)
			walletOperator := must(tvm.NewRandomV5R1TestWallet(_api, chainID))

			client := ton_mocks.NewAPIClientWrapped(t)

			if tt.mockSetup != nil {
				tt.mockSetup(_api, client)
			}

			executor, err := mcmston.NewTimelockExecutor(mcmston.TimelockExecutorOpts{
				Client: client,
				Wallet: walletOperator,
				Amount: tlb.MustFromTON("0.1"),
			})
			require.NoError(t, err)

			tx, err := executor.Execute(ctx, tt.bop, tt.timelockAddress, tt.predecessor, tt.salt)
			require.Equal(t, tt.wantTxHash, tx.Hash)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
