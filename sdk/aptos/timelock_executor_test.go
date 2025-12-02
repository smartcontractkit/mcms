package aptos

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"

	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	mock_aptossdk "github.com/smartcontractkit/mcms/sdk/aptos/mocks/aptos"
	mock_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms"
)

func TestNewTimelockExecutor(t *testing.T) {
	t.Parallel()
	mockClient := mock_aptossdk.NewAptosRpcClient(t)
	mockSigner := mock_aptossdk.NewTransactionSigner(t)
	executor := NewTimelockExecutor(mockClient, mockSigner)
	assert.Equal(t, mockClient, executor.client)
	assert.Equal(t, mockSigner, executor.auth)
	assert.NotNil(t, executor.bindingFn)
}

func TestExecutorExecute(t *testing.T) {
	t.Parallel()
	type args struct {
		bop             types.BatchOperation
		timelockAddress string
		predecessor     common.Hash
		salt            common.Hash
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(mcms *mock_mcms.MCMS)
		want      types.TransactionResult
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain5Selector,
					Transactions: []types.Transaction{
						{
							To:   "0x456",
							Data: []byte{0x12, 0x34},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package1",
								ModuleName:  "module1",
								Function:    "function_one",
							})),
						}, {
							To:   "0x789",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package2",
								ModuleName:  "module2",
								Function:    "function_two",
							})),
						},
					},
				},
				timelockAddress: "0x123",
				predecessor:     common.Hash{},
				salt:            common.HexToHash("0xabcd"),
			},
			mockSetup: func(mcms *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				mcms.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockExecuteBatch(
					mock.Anything,
					[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
					[]string{"module1", "module2"},
					[]string{"function_one", "function_two"},
					[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
				).Return(&api.PendingTransaction{Hash: "0xcoffee"}, nil)
			},
			want: types.TransactionResult{
				Hash:        "0xcoffee",
				ChainFamily: cselectors.FamilyAptos,
				RawData:     &api.PendingTransaction{Hash: "0xcoffee"},
			},
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid timelock address",
			args: args{
				timelockAddress: "invalidaddress!",
			},
			wantErr: AssertErrorContains("parse Timelock address"),
		}, {
			name: "failure - invalid additional fields",
			args: args{
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain5Selector,
					Transactions: []types.Transaction{
						{
							To:               "0x123",
							AdditionalFields: []byte("invalid json"),
						},
					},
				},
				timelockAddress: "0x123",
			},
			wantErr: AssertErrorContains("unmarshal additional fields"),
		}, {
			name: "failure - invalid To address",
			args: args{
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain5Selector,
					Transactions: []types.Transaction{
						{
							To: "invalidaddress!",
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package1",
								ModuleName:  "module1",
								Function:    "function_one",
							})),
						},
					},
				},
				timelockAddress: "0x123",
			},
			wantErr: AssertErrorContains("parse To address"),
		}, {
			name: "failure - TimelockExecuteBatch failed",
			args: args{
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain5Selector,
					Transactions: []types.Transaction{
						{
							To:   "0x456",
							Data: []byte{0x12, 0x34},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package1",
								ModuleName:  "module1",
								Function:    "function_one",
							})),
						}, {
							To:   "0x789",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package2",
								ModuleName:  "module2",
								Function:    "function_two",
							})),
						},
					},
				},
				timelockAddress: "0x123",
				predecessor:     common.Hash{},
				salt:            common.HexToHash("0xabcd"),
			},
			mockSetup: func(mcms *mock_mcms.MCMS) {
				mockMCMSModule := mock_module_mcms.NewMCMSInterface(t)
				mcms.EXPECT().MCMS().Return(mockMCMSModule)
				mockMCMSModule.EXPECT().TimelockExecuteBatch(
					mock.Anything,
					[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
					[]string{"module1", "module2"},
					[]string{"function_one", "function_two"},
					[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
				).Return(nil, errors.New("error during TimelockExecuteBatch"))
			},
			want:    types.TransactionResult{},
			wantErr: AssertErrorContains("error during TimelockExecuteBatch"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			executor := TimelockExecutor{
				bindingFn: func(_ aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			got, err := executor.Execute(t.Context(), tt.args.bop, tt.args.timelockAddress, tt.args.predecessor, tt.args.salt)
			if !tt.wantErr(t, err, fmt.Sprintf("Execute(%v, %v, %v, %v)", tt.args.bop, tt.args.timelockAddress, tt.args.predecessor, tt.args.salt)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
