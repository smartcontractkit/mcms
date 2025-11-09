package ton_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"

	tonmcms "github.com/smartcontractkit/mcms/sdk/ton"

	ton_mocks "github.com/smartcontractkit/mcms/sdk/ton/mocks"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	encoder := &tonmcms.Encoder{}
	chainID := chaintest.Chain7TONID

	_api := ton_mocks.NewTonAPI(t)
	walletOperator := must(makeRandomTestWallet(_api, chainID))
	client := ton_mocks.NewAPIClientWrapped(t)

	executor, err := tonmcms.NewExecutor(encoder, client, walletOperator, tlb.MustFromTON("0.1"))
	assert.NotNil(t, executor, "expected Executor")
	assert.NoError(t, err)
}

func TestExecutor_ExecuteOperation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name       string
		encoder    *tonmcms.Encoder
		metadata   types.ChainMetadata
		nonce      uint32
		proof      []common.Hash
		op         types.Operation
		mockSetup  func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped)
		wantTxHash string
		wantErrNew error
		wantErr    error
	}{
		{
			name: "success",
			encoder: &tonmcms.Encoder{
				ChainSelector: chaintest.Chain7Selector,
			},
			metadata: types.ChainMetadata{
				MCMAddress: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
			nonce: 1,
			op: types.Operation{
				ChainSelector: chaintest.Chain7Selector,
				Transaction: types.Transaction{
					To:               "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
					Data:             cell.BeginCell().MustStoreBinarySnake([]byte{1, 2, 3}).EndCell().ToBOC(),
					AdditionalFields: json.RawMessage(`{"value": 0}`)},
			},
			mockSetup: func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
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

				// Mock SendTransaction to return an error
				api.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, nil)
			},
			wantTxHash: "010203040e",
			wantErr:    nil,
		},
		{
			name: "failure in tx execution",
			encoder: &tonmcms.Encoder{
				ChainSelector: chaintest.Chain7Selector,
			},
			metadata: types.ChainMetadata{
				MCMAddress: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
			nonce: 1,
			op: types.Operation{
				ChainSelector: chaintest.Chain7Selector,
				Transaction: types.Transaction{
					To:               "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
					Data:             cell.BeginCell().MustStoreBinarySnake([]byte{1, 2, 3}).EndCell().ToBOC(),
					AdditionalFields: json.RawMessage(`{"value": 0}`)},
			},
			mockSetup: func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
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

				// Mock SendTransaction to return an error
				api.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, errors.New("error during tx send"))
			},
			wantTxHash: "",
			wantErr:    errors.New("failed to execute op: error during tx send"),
		},
		{
			name:       "failure - nil encoder",
			encoder:    nil,
			mockSetup:  func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {},
			wantTxHash: "",
			wantErrNew: errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil"),
		},
		{
			name: "failure in operation conversion due to invalid chain ID",
			encoder: &tonmcms.Encoder{
				ChainSelector: types.ChainSelector(1),
			},
			op: types.Operation{
				ChainSelector: types.ChainSelector(1),
				Transaction: types.Transaction{
					To:               "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
					Data:             cell.BeginCell().MustStoreBinarySnake([]byte{1, 2, 3}).EndCell().ToBOC(),
					AdditionalFields: json.RawMessage(`{"value": 0}`)},
			},
			mockSetup:  func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {},
			wantTxHash: "",
			wantErr:    errors.New("failed to convert to operation: invalid chain ID: 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Initialize the mock
			chainID := chaintest.Chain7TONID
			_api := ton_mocks.NewTonAPI(t)
			walletOperator := must(makeRandomTestWallet(_api, chainID))

			client := ton_mocks.NewAPIClientWrapped(t)

			if tt.mockSetup != nil {
				tt.mockSetup(_api, client)
			}

			executor, err := tonmcms.NewExecutor(tt.encoder, client, walletOperator, tlb.MustFromTON("0.1"))
			if tt.wantErrNew != nil {
				assert.EqualError(t, err, tt.wantErrNew.Error())
				return
			} else {
				assert.NoError(t, err)
			}

			tx, err := executor.ExecuteOperation(ctx, tt.metadata, tt.nonce, tt.proof, tt.op)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTxHash, tx.Hash)
			}
		})
	}
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name             string
		encoder          *tonmcms.Encoder
		metadata         types.ChainMetadata
		proof            []common.Hash
		root             [32]byte
		validUntil       uint32
		sortedSignatures []types.Signature
		mockSetup        func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped)
		wantTxHash       string
		wantErrNew       error
		wantErr          error
	}{
		{
			name: "success",
			encoder: &tonmcms.Encoder{
				ChainSelector: chaintest.Chain7Selector,
			},
			metadata: types.ChainMetadata{
				MCMAddress: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
			root:       [32]byte{1, 2, 3},
			validUntil: 4130013354,
			sortedSignatures: []types.Signature{
				makeTestSignature("0xabcdef1234567890"),
				makeTestSignature("0xabcdef1234567890"),
			},
			mockSetup: func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
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

				// Mock SendTransaction to return an error
				api.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, nil)
			},
			wantTxHash: "010203040e",
			wantErr:    nil,
		},
		{
			name: "failure in tx send",
			encoder: &tonmcms.Encoder{
				ChainSelector: chaintest.Chain7Selector,
			},
			metadata: types.ChainMetadata{
				MCMAddress: "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
			},
			root:       [32]byte{1, 2, 3},
			validUntil: 4130013354,
			sortedSignatures: []types.Signature{ // TODO: "failed to encode signatures: failed to recover public key: recovery failed"
				makeTestSignature("0xabcdef1234567890"),
				makeTestSignature("0xabcdef1234567890"),
			},
			mockSetup: func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {
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

				// Mock SendTransaction to return an error
				api.EXPECT().SendExternalMessageWaitTransaction(mock.Anything, mock.Anything).
					Return(&tlb.Transaction{Hash: []byte{1, 2, 3, 4, 14}}, &ton.BlockIDExt{}, []byte{}, errors.New("error during tx send"))
			},
			wantTxHash: "",
			wantErr:    errors.New("failed to set root: error during tx send"),
		},
		{
			name:       "failure - nil encoder",
			encoder:    nil,
			mockSetup:  func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {},
			wantTxHash: "",
			wantErrNew: errors.New("failed to create sdk.Executor - encoder (sdk.Encoder) is nil"),
		},
		{
			name: "failure in operation conversion due to invalid chain ID",
			encoder: &tonmcms.Encoder{
				ChainSelector: types.ChainSelector(1),
			},
			mockSetup:  func(api *ton_mocks.TonAPI, client *ton_mocks.APIClientWrapped) {},
			wantTxHash: "",
			wantErr:    errors.New("failed to convert to root metadata: invalid chain ID: 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Initialize the mock
			chainID := chaintest.Chain7TONID
			_api := ton_mocks.NewTonAPI(t)
			walletOperator := must(makeRandomTestWallet(_api, chainID))

			client := ton_mocks.NewAPIClientWrapped(t)

			if tt.mockSetup != nil {
				tt.mockSetup(_api, client)
			}

			executor, err := tonmcms.NewExecutor(tt.encoder, client, walletOperator, tlb.MustFromTON("0.1"))
			if tt.wantErrNew != nil {
				assert.EqualError(t, err, tt.wantErrNew.Error())
				return
			} else {
				assert.NoError(t, err)
			}

			tx, err := executor.SetRoot(ctx, tt.metadata,
				tt.proof,
				tt.root,
				tt.validUntil,
				tt.sortedSignatures)

			assert.Equal(t, tt.wantTxHash, tx.Hash)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func makeTestSignature(hexStr string) types.Signature {
	// Private key to use for signing
	pk, _ := crypto.GenerateKey()

	// Hash to sign
	hash := common.HexToHash(hexStr)
	sigBytes, _ := crypto.Sign(hash.Bytes(), pk)

	// Signature object for the hash
	sig, _ := types.NewSignatureFromBytes(sigBytes)
	return sig
}
