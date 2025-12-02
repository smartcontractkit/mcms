package aptos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	mock_aptossdk "github.com/smartcontractkit/mcms/sdk/aptos/mocks/aptos"
	mock_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms"
	mock_module_mcms_executor "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms_executor"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()
	encoder := NewEncoder(chaintest.Chain5Selector, 1, true)
	mockClient := mock_aptossdk.NewAptosRpcClient(t)
	mockSigner := mock_aptossdk.NewTransactionSigner(t)
	executor := NewExecutor(mockClient, mockSigner, encoder, TimelockRoleProposer)
	assert.Equal(t, mockClient, executor.client)
	assert.Equal(t, mockSigner, executor.auth)
	assert.Equal(t, encoder, executor.Encoder)
	assert.Equal(t, TimelockRoleProposer, executor.role)
}

func TestExecutorExecuteOperation(t *testing.T) {
	t.Parallel()
	generateData := func(length int) []byte {
		return bytes.Repeat([]byte{0x42}, length)
	}
	type args struct {
		metadata types.ChainMetadata
		nonce    uint32
		proof    []common.Hash
		op       types.Operation
	}
	tests := []struct {
		name          string
		chainSelector types.ChainSelector
		args          args
		mockSetup     func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS)
		want          types.TransactionResult
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			name:          "success",
			chainSelector: chaintest.Chain5Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					AdditionalFields: Must(json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleProposer})),
				},
				nonce: 42,
				proof: []common.Hash{common.HexToHash("0x123456789")},
				op: types.Operation{
					ChainSelector: 0,
					Transaction: types.Transaction{
						To:               "0x456",
						Data:             []byte{0x12, 0x34, 0x56, 0x78},
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			mockSetup: func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				mcms.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().Execute(
					mock.Anything,
					TimelockRoleProposer.Byte(),
					new(big.Int).SetUint64(chaintest.Chain5AptosID),
					Must(hexToAddress("0x123")),
					uint64(42),
					Must(hexToAddress("0x456")),
					"module",
					"function_one",
					[]byte{0x12, 0x34, 0x56, 0x78},
					[][]byte{common.HexToHash("0x123456789").Bytes()},
				).Return(&api.PendingTransaction{Hash: "0xdeadbeef"}, nil)
			},
			want: types.TransactionResult{
				Hash:        "0xdeadbeef",
				ChainFamily: cselectors.FamilyAptos,
				RawData:     &api.PendingTransaction{Hash: "0xdeadbeef"},
			},
			wantErr: assert.NoError,
		}, {
			name:          "success - chunked data",
			chainSelector: chaintest.Chain5Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					AdditionalFields: Must(json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleProposer})),
				},
				nonce: 42,
				proof: []common.Hash{common.HexToHash("0x123456789")},
				op: types.Operation{
					ChainSelector: 0,
					Transaction: types.Transaction{
						To:               "0x456",
						Data:             generateData(120_000),
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			mockSetup: func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS) {
				signer.EXPECT().AccountAddress().Return(Must(hexToAddress("0x111111")))
				client.EXPECT().Account(Must(hexToAddress("0x111111"))).Return(aptos.AccountInfo{
					SequenceNumberStr: "789",
				}, nil)
				mockMCMSExecutorModule := mock_module_mcms_executor.NewMCMSExecutorInterface(t)
				mcms.EXPECT().MCMSExecutor().Return(mockMCMSExecutorModule)
				mockMCMSExecutorModule.EXPECT().StageData(
					&bind.TransactOpts{
						SequenceNumber: pointerTo(uint64(789)),
						Signer:         signer,
					},
					generateData(50_000),
					mock.Anything,
				).Return(&api.PendingTransaction{Hash: "0xdeadbeef1"}, nil)
				mockMCMSExecutorModule.EXPECT().StageData(
					&bind.TransactOpts{
						SequenceNumber: pointerTo(uint64(790)),
						Signer:         signer,
					},
					generateData(50_000),
					mock.Anything,
				).Return(&api.PendingTransaction{Hash: "0xdeadbeef2"}, nil)
				mockMCMSExecutorModule.EXPECT().StageDataAndExecute(
					&bind.TransactOpts{
						SequenceNumber: pointerTo(uint64(791)),
						Signer:         signer,
					},
					TimelockRoleProposer.Byte(),
					new(big.Int).SetUint64(chaintest.Chain5AptosID),
					Must(hexToAddress("0x123")),
					uint64(42),
					Must(hexToAddress("0x456")),
					"module",
					"function_one",
					generateData(20_000),
					[][]byte{common.HexToHash("0x123456789").Bytes()},
				).Return(&api.PendingTransaction{Hash: "0xdeadbeef3"}, nil)
			},
			want: types.TransactionResult{
				Hash:        "0xdeadbeef3",
				ChainFamily: cselectors.FamilyAptos,
				RawData:     &api.PendingTransaction{Hash: "0xdeadbeef3"},
			},
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "invalidaddress",
				},
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name: "failure - invalid to address",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "0x123",
				},
				op: types.Operation{
					Transaction: types.Transaction{
						To: "invalidaddress",
					},
				},
			},
			wantErr: AssertErrorContains("parse To address"),
		}, {
			name: "failure - invalid additional fields",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "0x123",
				},
				op: types.Operation{
					Transaction: types.Transaction{
						To:               "0x111",
						AdditionalFields: []byte("invalid json"),
					},
				},
			},
			wantErr: AssertErrorContains("unmarshal additional fields"),
		}, {
			name: "failure - invalid additional fields metadata",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					AdditionalFields: []byte("invalid json"),
				},
				op: types.Operation{
					Transaction: types.Transaction{
						To:               "0x111",
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			wantErr: AssertErrorContains("unmarshal additional fields metadata"),
		}, {
			name:          "failure - invalid chain selector",
			chainSelector: chaintest.Chain1Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "0x123",
				},
				op: types.Operation{
					Transaction: types.Transaction{
						To:               "0x111",
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			wantErr: AssertErrorContains("not found for selector"),
		}, {
			name:          "failure - Execute failed",
			chainSelector: chaintest.Chain5Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					AdditionalFields: Must(json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleCanceller})),
				},
				nonce: 42,
				proof: []common.Hash{common.HexToHash("0x123456789")},
				op: types.Operation{
					ChainSelector: 0,
					Transaction: types.Transaction{
						To:               "0x456",
						Data:             []byte{0x12, 0x34, 0x56, 0x78},
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			mockSetup: func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				mcms.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().Execute(
					mock.Anything,
					TimelockRoleCanceller.Byte(),
					new(big.Int).SetUint64(chaintest.Chain5AptosID),
					Must(hexToAddress("0x123")),
					uint64(42),
					Must(hexToAddress("0x456")),
					"module",
					"function_one",
					[]byte{0x12, 0x34, 0x56, 0x78},
					[][]byte{common.HexToHash("0x123456789").Bytes()},
				).Return(nil, errors.New("error during Execute"))
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("error during Execute"),
		}, {
			name:          "failure - Account() failed",
			chainSelector: chaintest.Chain5Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "0x123",
				},
				nonce: 42,
				proof: []common.Hash{common.HexToHash("0x123456789")},
				op: types.Operation{
					ChainSelector: 0,
					Transaction: types.Transaction{
						To:               "0x456",
						Data:             generateData(120_000),
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			mockSetup: func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS) {
				signer.EXPECT().AccountAddress().Return(Must(hexToAddress("0x111111")))
				client.EXPECT().Account(Must(hexToAddress("0x111111"))).Return(aptos.AccountInfo{}, errors.New("error during Account"))
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("error during Account"),
		}, {
			name:          "failure - StageData() failed",
			chainSelector: chaintest.Chain5Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "0x123",
				},
				nonce: 42,
				proof: []common.Hash{common.HexToHash("0x123456789")},
				op: types.Operation{
					ChainSelector: 0,
					Transaction: types.Transaction{
						To:               "0x456",
						Data:             generateData(75_000),
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			mockSetup: func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS) {
				signer.EXPECT().AccountAddress().Return(Must(hexToAddress("0x111111")))
				client.EXPECT().Account(Must(hexToAddress("0x111111"))).Return(aptos.AccountInfo{
					SequenceNumberStr: "789",
				}, nil)
				mockMCMSExecutorModule := mock_module_mcms_executor.NewMCMSExecutorInterface(t)
				mcms.EXPECT().MCMSExecutor().Return(mockMCMSExecutorModule)
				mockMCMSExecutorModule.EXPECT().StageData(
					&bind.TransactOpts{
						SequenceNumber: pointerTo(uint64(789)),
						Signer:         signer,
					},
					generateData(50_000),
					mock.Anything,
				).Return(nil, errors.New("error during StageData"))
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("error during StageData"),
		}, {
			name:          "failure - StageDataAndExecute() failed",
			chainSelector: chaintest.Chain5Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					AdditionalFields: Must(json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleBypasser})),
				},
				nonce: 42,
				proof: []common.Hash{common.HexToHash("0x123456789")},
				op: types.Operation{
					ChainSelector: 0,
					Transaction: types.Transaction{
						To:               "0x456",
						Data:             generateData(72_000),
						AdditionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function_one"}`),
					},
				},
			},
			mockSetup: func(client *mock_aptossdk.AptosRpcClient, signer *mock_aptossdk.TransactionSigner, mcms *mock_mcms.MCMS) {
				signer.EXPECT().AccountAddress().Return(Must(hexToAddress("0x111111")))
				client.EXPECT().Account(Must(hexToAddress("0x111111"))).Return(aptos.AccountInfo{
					SequenceNumberStr: "789",
				}, nil)
				mockMCMSExecutorModule := mock_module_mcms_executor.NewMCMSExecutorInterface(t)
				mcms.EXPECT().MCMSExecutor().Return(mockMCMSExecutorModule)
				mockMCMSExecutorModule.EXPECT().StageData(
					&bind.TransactOpts{
						SequenceNumber: pointerTo(uint64(789)),
						Signer:         signer,
					},
					generateData(50_000),
					mock.Anything,
				).Return(&api.PendingTransaction{Hash: "0xdeadbeef1"}, nil)
				mockMCMSExecutorModule.EXPECT().StageDataAndExecute(
					&bind.TransactOpts{
						SequenceNumber: pointerTo(uint64(790)),
						Signer:         signer,
					},
					TimelockRoleBypasser.Byte(),
					new(big.Int).SetUint64(chaintest.Chain5AptosID),
					Must(hexToAddress("0x123")),
					uint64(42),
					Must(hexToAddress("0x456")),
					"module",
					"function_one",
					generateData(22_000),
					[][]byte{common.HexToHash("0x123456789").Bytes()},
				).Return(nil, errors.New("error during StageDataAndExecute"))
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("error during StageDataAndExecute"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mock_aptossdk.NewAptosRpcClient(t)
			mockSigner := mock_aptossdk.NewTransactionSigner(t)
			mcmsBinding := mock_mcms.NewMCMS(t)
			executor := Executor{
				Encoder: NewEncoder(tt.chainSelector, 0, false),
				client:  mockClient,
				auth:    mockSigner,
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mockClient, mockSigner, mcmsBinding)
			}

			got, err := executor.ExecuteOperation(t.Context(), tt.args.metadata, tt.args.nonce, tt.args.proof, tt.args.op)
			if !tt.wantErr(t, err, fmt.Sprintf("ExecuteOperation(%v, %v, %v, %v)", tt.args.metadata, tt.args.nonce, tt.args.proof, tt.args.op)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExecutorSetRoot(t *testing.T) {
	t.Parallel()
	type args struct {
		metadata         types.ChainMetadata
		proof            []common.Hash
		root             [32]byte
		validUntil       uint32
		sortedSignatures []types.Signature
	}
	tests := []struct {
		name                 string
		chainSelector        types.ChainSelector
		txCount              uint64
		overridePreviousRoot bool
		args                 args
		mockSetup            func(mcms *mock_mcms.MCMS)
		want                 types.TransactionResult
		wantErr              assert.ErrorAssertionFunc
	}{
		{
			name:                 "success",
			chainSelector:        chaintest.Chain5Selector,
			txCount:              33,
			overridePreviousRoot: true,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					StartingOpCount:  45,
					AdditionalFields: Must(json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleProposer})),
				},
				proof:      []common.Hash{common.HexToHash("0x123456789")},
				root:       [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 23, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				validUntil: 1742987131,
				sortedSignatures: []types.Signature{
					{},
					{},
				},
			},
			mockSetup: func(mcms *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				mcms.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().SetRoot(
					mock.Anything,
					TimelockRoleProposer.Byte(),
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 23, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
					uint64(1742987131),
					new(big.Int).SetUint64(chaintest.Chain5AptosID),
					Must(hexToAddress("0x123")),
					uint64(45),
					uint64(78),
					true,
					[][]byte{common.HexToHash("0x123456789").Bytes()},
					mock.Anything,
				).Return(&api.PendingTransaction{Hash: "0x111111"}, nil)
			},
			want: types.TransactionResult{
				Hash:        "0x111111",
				ChainFamily: cselectors.FamilyAptos,
				RawData:     &api.PendingTransaction{Hash: "0x111111"},
			},
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid MCMS address",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "invalidaddress",
				},
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name:          "failure - invalid additional fields metadata",
			chainSelector: chaintest.Chain2Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x1",
					AdditionalFields: []byte("invalid json"),
				},
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("unmarshal additional fields metadata"),
		}, {
			name:          "failure - invalid chain selector",
			chainSelector: chaintest.Chain2Selector,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "0x1",
				},
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("not found for selector"),
		}, {
			name:                 "failure - SetRoot() failed",
			chainSelector:        chaintest.Chain5Selector,
			txCount:              33,
			overridePreviousRoot: true,
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "0x123",
					StartingOpCount:  45,
					AdditionalFields: Must(json.Marshal(AdditionalFieldsMetadata{Role: TimelockRoleCanceller})),
				},
				proof:      []common.Hash{common.HexToHash("0x123456789")},
				root:       [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 23, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				validUntil: 1742987131,
				sortedSignatures: []types.Signature{
					{V: 65},
					{},
				},
			},
			mockSetup: func(mcms *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				mcms.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().SetRoot(
					mock.Anything,
					TimelockRoleCanceller.Byte(),
					[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 23, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
					uint64(1742987131),
					new(big.Int).SetUint64(chaintest.Chain5AptosID),
					Must(hexToAddress("0x123")),
					uint64(45),
					uint64(78),
					true,
					[][]byte{common.HexToHash("0x123456789").Bytes()},
					mock.Anything,
				).Return(nil, errors.New("error during SetRoot"))
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("error during SetRoot"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			executor := Executor{
				Encoder: NewEncoder(tt.chainSelector, tt.txCount, tt.overridePreviousRoot),
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := executor.SetRoot(context.Background(), tt.args.metadata, tt.args.proof, tt.args.root, tt.args.validUntil, tt.args.sortedSignatures)
			if !tt.wantErr(t, err, fmt.Sprintf("SetRoot(%v, %v, %v, %v, %v)", tt.args.metadata, tt.args.proof, tt.args.root, tt.args.validUntil, tt.args.sortedSignatures)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
