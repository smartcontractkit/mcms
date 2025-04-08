package aptos

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	mock_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms"
	mock_module_mcms "github.com/smartcontractkit/mcms/sdk/aptos/mocks/mcms/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func TestNewTimelockConverter(t *testing.T) {
	t.Parallel()
	converter := NewTimelockConverter()
	assert.NotNil(t, converter)
	assert.NotNil(t, converter.bindingFn)
}

func TestTimelockConverter_ConvertBatchToChainOperations(t *testing.T) {
	t.Parallel()
	type args struct {
		metadata        types.ChainMetadata
		bop             types.BatchOperation
		timelockAddress string
		mcmAddress      string
		delay           types.Duration
		action          types.TimelockAction
		predecessor     common.Hash
		salt            common.Hash
	}
	tests := []struct {
		name           string
		args           args
		mockSetup      func(m *mock_mcms.MCMS)
		wantOperations []types.Operation
		wantHash       common.Hash
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "success - schedule",
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
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag1", "tag2"},
							},
						}, {
							To:   "0x789",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package2",
								ModuleName:  "module2",
								Function:    "function_two",
							})),
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag3", "tag4"},
							},
						},
					},
				},
				mcmAddress:  "0x123",
				delay:       types.NewDuration(time.Second * 42),
				action:      types.TimelockActionSchedule,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMS := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMS)
				mockMCMSEncoder := mock_module_mcms.NewMCMSEncoder(t)
				mockMCMS.EXPECT().Encoder().Return(mockMCMSEncoder)
				mockMCMSEncoder.EXPECT().TimelockScheduleBatch(
					[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
					[]string{"module1", "module2"},
					[]string{"function_one", "function_two"},
					[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
					uint64(42),
				).Return(
					bind.ModuleInformation{
						PackageName: "mcms",
						ModuleName:  "mcms_timelock",
					},
					"timelock_schedule_batch",
					nil,
					[][]byte{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
					nil,
				)
			},
			wantOperations: []types.Operation{
				{
					ChainSelector: chaintest.Chain5Selector,
					Transaction: types.Transaction{
						OperationMetadata: types.OperationMetadata{
							ContractType: "MCMSTimelock",
							Tags:         []string{"tag1", "tag2", "tag3", "tag4"},
						},
						To:   mustHexToAddress("0x123").StringLong(),
						Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
						AdditionalFields: Must(json.Marshal(AdditionalFields{
							PackageName: "mcms",
							ModuleName:  "mcms_timelock",
							Function:    "timelock_schedule_batch",
						})),
					},
				},
			},
			wantHash: Must(HashOperationBatch(
				[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
				[]string{"module1", "module2"},
				[]string{"function_one", "function_two"},
				[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
			)),
			wantErr: assert.NoError,
		},
		{
			name: "success - bypass",
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
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag1", "tag2"},
							},
						}, {
							To:   "0x789",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package2",
								ModuleName:  "module2",
								Function:    "function_two",
							})),
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag3", "tag4"},
							},
						},
					},
				},
				mcmAddress:  "0x123",
				delay:       types.NewDuration(time.Second * 42),
				action:      types.TimelockActionBypass,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMS := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMS)
				mockMCMSEncoder := mock_module_mcms.NewMCMSEncoder(t)
				mockMCMS.EXPECT().Encoder().Return(mockMCMSEncoder)
				mockMCMSEncoder.EXPECT().TimelockBypasserExecuteBatch(
					[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
					[]string{"module1", "module2"},
					[]string{"function_one", "function_two"},
					[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
				).Return(
					bind.ModuleInformation{
						PackageName: "mcms",
						ModuleName:  "mcms_timelock",
					},
					"timelock_bypasser_execute_batch",
					nil,
					[][]byte{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
					nil,
				)
			},
			wantOperations: []types.Operation{
				{
					ChainSelector: chaintest.Chain5Selector,
					Transaction: types.Transaction{
						OperationMetadata: types.OperationMetadata{
							ContractType: "MCMSTimelock",
							Tags:         []string{"tag1", "tag2", "tag3", "tag4"},
						},
						To:   mustHexToAddress("0x123").StringLong(),
						Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
						AdditionalFields: Must(json.Marshal(AdditionalFields{
							PackageName: "mcms",
							ModuleName:  "mcms_timelock",
							Function:    "timelock_bypasser_execute_batch",
						})),
					},
				},
			},
			wantHash: Must(HashOperationBatch(
				[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
				[]string{"module1", "module2"},
				[]string{"function_one", "function_two"},
				[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
			)),
			wantErr: assert.NoError,
		},
		{
			name: "success - cancel",
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
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag1", "tag2"},
							},
						}, {
							To:   "0x789",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package2",
								ModuleName:  "module2",
								Function:    "function_two",
							})),
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag3", "tag4"},
							},
						},
					},
				},
				mcmAddress:  "0x123",
				delay:       types.NewDuration(time.Second * 42),
				action:      types.TimelockActionCancel,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMS := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMS)
				mockMCMSEncoder := mock_module_mcms.NewMCMSEncoder(t)
				mockMCMS.EXPECT().Encoder().Return(mockMCMSEncoder)
				mockMCMSEncoder.EXPECT().TimelockCancel(
					Must(HashOperationBatch(
						[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
						[]string{"module1", "module2"},
						[]string{"function_one", "function_two"},
						[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
						[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
					)).Bytes(),
				).Return(
					bind.ModuleInformation{
						PackageName: "mcms",
						ModuleName:  "mcms_timelock",
					},
					"timelock_cancel",
					nil,
					[][]byte{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
					nil,
				)
			},
			wantOperations: []types.Operation{
				{
					ChainSelector: chaintest.Chain5Selector,
					Transaction: types.Transaction{
						OperationMetadata: types.OperationMetadata{
							ContractType: "MCMSTimelock",
							Tags:         []string{"tag1", "tag2", "tag3", "tag4"},
						},
						To:   mustHexToAddress("0x123").StringLong(),
						Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
						AdditionalFields: Must(json.Marshal(AdditionalFields{
							PackageName: "mcms",
							ModuleName:  "mcms_timelock",
							Function:    "timelock_cancel",
						})),
					},
				},
			},
			wantHash: Must(HashOperationBatch(
				[]aptos.AccountAddress{Must(hexToAddress("0x456")), Must(hexToAddress("0x789"))},
				[]string{"module1", "module2"},
				[]string{"function_one", "function_two"},
				[][]byte{{0x12, 0x34}, {0xab, 0xcd}},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xab, 0xcd},
			)),
			wantErr: assert.NoError,
		},
		{
			name: "failure - invalid MCMS address",
			args: args{
				mcmAddress: "invalid address",
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		},
		{
			name: "failure - invalid additional fields",
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
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag1", "tag2"},
							},
						}, {
							To:               "0x789",
							Data:             []byte{0xab, 0xcd},
							AdditionalFields: []byte("invalid json!"),
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag3", "tag4"},
							},
						},
					},
				},
				mcmAddress: "0x123",
			},
			wantErr: AssertErrorContains("unmarshal additional fields"),
		},
		{
			name: "failure - invalid To address",
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
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag1", "tag2"},
							},
						}, {
							To:   "invalid address 0x123",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: Must(json.Marshal(AdditionalFields{
								PackageName: "package2",
								ModuleName:  "module2",
								Function:    "function_two",
							})),
							OperationMetadata: types.OperationMetadata{
								Tags: []string{"tag3", "tag4"},
							},
						},
					},
				},
				mcmAddress: "0x123",
			},
			wantErr: AssertErrorContains("parse To address"),
		},
		{
			name: "failure - TimelockScheduleBatch failed",
			args: args{
				mcmAddress: "0x123",
				action:     types.TimelockActionSchedule,
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMS := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMS)
				mockMCMSEncoder := mock_module_mcms.NewMCMSEncoder(t)
				mockMCMS.EXPECT().Encoder().Return(mockMCMSEncoder)
				mockMCMSEncoder.EXPECT().TimelockScheduleBatch(
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(
					bind.ModuleInformation{}, "", nil, nil,
					errors.New("error during TimelockScheduleBatch"),
				)
			},
			wantErr: AssertErrorContains("error during TimelockScheduleBatch"),
		}, {
			name: "failure - TimelockScheduleBatch failed",
			args: args{
				mcmAddress: "0x123",
				action:     types.TimelockActionBypass,
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMS := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMS)
				mockMCMSEncoder := mock_module_mcms.NewMCMSEncoder(t)
				mockMCMS.EXPECT().Encoder().Return(mockMCMSEncoder)
				mockMCMSEncoder.EXPECT().TimelockBypasserExecuteBatch(
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(
					bind.ModuleInformation{}, "", nil, nil,
					errors.New("error during TimelockBypasserExecuteBatch"),
				)
			},
			wantErr: AssertErrorContains("error during TimelockBypasserExecuteBatch"),
		}, {
			name: "failure - TimelockCancel failed",
			args: args{
				mcmAddress: "0x123",
				action:     types.TimelockActionCancel,
			},
			mockSetup: func(m *mock_mcms.MCMS) {
				mockMCMS := mock_module_mcms.NewMCMS(t)
				m.EXPECT().MCMS().Return(mockMCMS)
				mockMCMSEncoder := mock_module_mcms.NewMCMSEncoder(t)
				mockMCMS.EXPECT().Encoder().Return(mockMCMSEncoder)
				mockMCMSEncoder.EXPECT().TimelockCancel(
					mock.Anything,
				).Return(
					bind.ModuleInformation{}, "", nil, nil,
					errors.New("error during TimelockCancel"),
				)
			},
			wantErr: AssertErrorContains("error during TimelockCancel"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mcmsBinding := mock_mcms.NewMCMS(t)
			converter := TimelockConverter{
				bindingFn: func(mcmsAddress aptos.AccountAddress, _ aptos.AptosRpcClient) mcms.MCMS {
					require.Equal(t, Must(hexToAddress(tt.args.mcmAddress)), mcmsAddress)
					return mcmsBinding
				},
			}

			if tt.mockSetup != nil {
				tt.mockSetup(mcmsBinding)
			}

			gotOperations, gotHash, err := converter.ConvertBatchToChainOperations(t.Context(), tt.args.metadata, tt.args.bop, tt.args.timelockAddress, tt.args.mcmAddress, tt.args.delay, tt.args.action, tt.args.predecessor, tt.args.salt)
			if !tt.wantErr(t, err, fmt.Sprintf("ConvertBatchToChainOperations(%v, %v, %v, %v, %v, %v, %v, %v)", tt.args.metadata, tt.args.bop, tt.args.timelockAddress, tt.args.mcmAddress, tt.args.delay, tt.args.action, tt.args.predecessor, tt.args.salt)) {
				return
			}
			assert.Equal(t, tt.wantOperations, gotOperations)
			assert.Equal(t, tt.wantHash, gotHash)
		})
	}
}

func TestHashOperationBatch(t *testing.T) {
	t.Parallel()
	// 0x860ab27255ad63f4b1cd56ffeb41953ba4b23d4d1f21e8e821ae7c1d4b0c8001

	targets := []aptos.AccountAddress{aptos.AccountOne}
	moduleNames := []string{"module"}
	functionNames := []string{"function"}
	datas := [][]byte{[]byte("asdf")}
	predecessor := []byte{1, 2, 3, 4}
	salt := []byte{5, 6, 7, 8}

	hash, err := HashOperationBatch(targets, moduleNames, functionNames, datas, predecessor, salt)
	require.NoError(t, err)
	require.Equal(t, "0x860ab27255ad63f4b1cd56ffeb41953ba4b23d4d1f21e8e821ae7c1d4b0c8001", hash.Hex())
}
