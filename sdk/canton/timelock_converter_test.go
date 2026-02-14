package canton

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"
)

// AssertErrorContains returns an error assertion function that checks for substring.
func AssertErrorContains(errorMessage string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...any) bool {
		return assert.ErrorContains(t, err, errorMessage, i...)
	}
}

// mustMarshalJSON marshals v to JSON and panics on error.
func mustMarshalJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func TestNewTimelockConverter(t *testing.T) {
	t.Parallel()
	converter := NewTimelockConverter()
	assert.NotNil(t, converter)
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
		name       string
		args       args
		wantOpLen  int
		wantErr    assert.ErrorAssertionFunc
		verifyHash bool
	}{
		{
			name: "success - schedule",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:              1337,
						MultisigId:           "test-multisig",
						InstanceId:           "test-instance",
						PreOpCount:           0,
						PostOpCount:          1,
						OverridePreviousRoot: false,
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "target-contract-1",
							Data: []byte{0x12, 0x34},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance-1",
								FunctionName:     "TestFunction",
								OperationData:    "abcd",
							}),
						},
					},
				},
				mcmAddress:  "mcms-contract-id-123",
				delay:       types.NewDuration(time.Second * 60),
				action:      types.TimelockActionSchedule,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			wantOpLen:  1,
			wantErr:    assert.NoError,
			verifyHash: true,
		},
		{
			name: "success - bypass",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "target-contract-1",
							Data: []byte{0x12, 0x34},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance-1",
								FunctionName:     "TestFunction",
								OperationData:    "abcd",
							}),
						},
					},
				},
				mcmAddress:  "mcms-contract-id-123",
				delay:       types.NewDuration(time.Second * 60),
				action:      types.TimelockActionBypass,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			wantOpLen:  1,
			wantErr:    assert.NoError,
			verifyHash: true,
		},
		{
			name: "success - cancel",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "target-contract-1",
							Data: []byte{0x12, 0x34},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance-1",
								FunctionName:     "TestFunction",
								OperationData:    "abcd",
							}),
						},
					},
				},
				mcmAddress:  "mcms-contract-id-123",
				delay:       types.NewDuration(time.Second * 60),
				action:      types.TimelockActionCancel,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			wantOpLen:  1,
			wantErr:    assert.NoError,
			verifyHash: true,
		},
		{
			name: "success - multiple transactions",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "target-contract-1",
							Data: []byte{0x12, 0x34},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance-1",
								FunctionName:     "Function1",
								OperationData:    "1234",
							}),
						},
						{
							To:   "target-contract-2",
							Data: []byte{0xab, 0xcd},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance-2",
								FunctionName:     "Function2",
								OperationData:    "5678",
							}),
						},
					},
				},
				mcmAddress:  "mcms-contract-id-123",
				delay:       types.NewDuration(time.Second * 120),
				action:      types.TimelockActionSchedule,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xef01"),
			},
			wantOpLen:  1,
			wantErr:    assert.NoError,
			verifyHash: true,
		},
		{
			name: "failure - unsupported action",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "target-contract-1",
							Data: []byte{0x12, 0x34},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance-1",
								FunctionName:     "TestFunction",
							}),
						},
					},
				},
				mcmAddress: "mcms-contract-id-123",
				action:     types.TimelockAction("unsupported"),
			},
			wantErr: AssertErrorContains("unsupported timelock action"),
		},
		{
			name: "failure - invalid metadata additional fields",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "mcms-contract-id-123",
					AdditionalFields: []byte("invalid json"),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions:  []types.Transaction{},
				},
				mcmAddress: "mcms-contract-id-123",
				action:     types.TimelockActionSchedule,
			},
			wantErr: AssertErrorContains("unmarshal metadata additional fields"),
		},
		{
			name: "failure - invalid transaction additional fields",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:               "target-contract-1",
							Data:             []byte{0x12, 0x34},
							AdditionalFields: []byte("invalid json"),
						},
					},
				},
				mcmAddress: "mcms-contract-id-123",
				action:     types.TimelockActionSchedule,
			},
			wantErr: AssertErrorContains("unmarshal transaction additional fields"),
		},
		{
			name: "success - fallback to tx.To when TargetInstanceId empty",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "fallback-target",
							Data: []byte{0x12, 0x34},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								FunctionName:  "TestFunction",
								OperationData: "abcd",
							}),
						},
					},
				},
				mcmAddress:  "mcms-contract-id-123",
				delay:       types.NewDuration(time.Second * 60),
				action:      types.TimelockActionSchedule,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			wantOpLen: 1,
			wantErr:   assert.NoError,
		},
		{
			name: "success - hex encode tx.Data when OperationData empty",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "mcms-contract-id-123",
					AdditionalFields: mustMarshalJSON(AdditionalFieldsMetadata{
						ChainId:    1337,
						MultisigId: "test-multisig",
						InstanceId: "test-instance",
					}),
				},
				bop: types.BatchOperation{
					ChainSelector: chaintest.Chain1Selector,
					Transactions: []types.Transaction{
						{
							To:   "target",
							Data: []byte{0xde, 0xad, 0xbe, 0xef},
							AdditionalFields: mustMarshalJSON(AdditionalFields{
								TargetInstanceId: "target-instance",
								FunctionName:     "TestFunction",
								// OperationData intentionally empty
							}),
						},
					},
				},
				mcmAddress:  "mcms-contract-id-123",
				delay:       types.NewDuration(time.Second * 60),
				action:      types.TimelockActionSchedule,
				predecessor: common.Hash{},
				salt:        common.HexToHash("0xabcd"),
			},
			wantOpLen: 1,
			wantErr:   assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter := NewTimelockConverter()

			gotOperations, gotHash, err := converter.ConvertBatchToChainOperations(
				context.Background(),
				tt.args.metadata,
				tt.args.bop,
				tt.args.timelockAddress,
				tt.args.mcmAddress,
				tt.args.delay,
				tt.args.action,
				tt.args.predecessor,
				tt.args.salt,
			)

			if !tt.wantErr(t, err, fmt.Sprintf("ConvertBatchToChainOperations(%v, %v, %v, %v, %v, %v, %v, %v)",
				tt.args.metadata, tt.args.bop, tt.args.timelockAddress, tt.args.mcmAddress,
				tt.args.delay, tt.args.action, tt.args.predecessor, tt.args.salt)) {
				return
			}

			if err == nil {
				assert.Len(t, gotOperations, tt.wantOpLen)
				if tt.verifyHash {
					assert.NotEqual(t, common.Hash{}, gotHash, "operation hash should not be empty")
				}

				// Verify operation structure
				if tt.wantOpLen > 0 {
					op := gotOperations[0]
					assert.Equal(t, tt.args.bop.ChainSelector, op.ChainSelector)
					assert.Equal(t, tt.args.mcmAddress, op.Transaction.To)
					assert.NotEmpty(t, op.Transaction.AdditionalFields)

					// Verify additional fields contain expected function name
					var opFields AdditionalFields
					err := json.Unmarshal(op.Transaction.AdditionalFields, &opFields)
					require.NoError(t, err)

					switch tt.args.action {
					case types.TimelockActionSchedule:
						assert.Equal(t, "ScheduleBatch", opFields.FunctionName)
					case types.TimelockActionBypass:
						assert.Equal(t, "BypasserExecuteBatch", opFields.FunctionName)
					case types.TimelockActionCancel:
						assert.Equal(t, "CancelBatch", opFields.FunctionName)
					}
				}
			}
		})
	}
}

func TestHashTimelockOpId(t *testing.T) {
	t.Parallel()

	calls := []TimelockCall{
		{
			TargetInstanceId: "target-1",
			FunctionName:     "function1",
			OperationData:    "abcd",
		},
	}
	predecessor := "0000000000000000000000000000000000000000000000000000000000000000"
	salt := "0000000000000000000000000000000000000000000000000000000000001234"

	hash := HashTimelockOpId(calls, predecessor, salt)
	assert.NotEqual(t, common.Hash{}, hash, "hash should not be empty")
}

func TestHashTimelockOpId_DifferentInputs(t *testing.T) {
	t.Parallel()

	predecessor := "0000000000000000000000000000000000000000000000000000000000000000"
	salt := "0000000000000000000000000000000000000000000000000000000000001234"

	// Same inputs should produce same hash
	calls1 := []TimelockCall{
		{TargetInstanceId: "target", FunctionName: "func", OperationData: "1234"},
	}
	calls2 := []TimelockCall{
		{TargetInstanceId: "target", FunctionName: "func", OperationData: "1234"},
	}
	hash1 := HashTimelockOpId(calls1, predecessor, salt)
	hash2 := HashTimelockOpId(calls2, predecessor, salt)
	assert.Equal(t, hash1, hash2, "same inputs should produce same hash")

	// Different target should produce different hash
	calls3 := []TimelockCall{
		{TargetInstanceId: "different-target", FunctionName: "func", OperationData: "1234"},
	}
	hash3 := HashTimelockOpId(calls3, predecessor, salt)
	assert.NotEqual(t, hash1, hash3, "different target should produce different hash")

	// Different function should produce different hash
	calls4 := []TimelockCall{
		{TargetInstanceId: "target", FunctionName: "different-func", OperationData: "1234"},
	}
	hash4 := HashTimelockOpId(calls4, predecessor, salt)
	assert.NotEqual(t, hash1, hash4, "different function should produce different hash")

	// Different operation data should produce different hash
	calls5 := []TimelockCall{
		{TargetInstanceId: "target", FunctionName: "func", OperationData: "5678"},
	}
	hash5 := HashTimelockOpId(calls5, predecessor, salt)
	assert.NotEqual(t, hash1, hash5, "different operation data should produce different hash")

	// Different salt should produce different hash
	differentSalt := "0000000000000000000000000000000000000000000000000000000000005678"
	hash6 := HashTimelockOpId(calls1, predecessor, differentSalt)
	assert.NotEqual(t, hash1, hash6, "different salt should produce different hash")

	// Different predecessor should produce different hash
	differentPredecessor := "1111111111111111111111111111111111111111111111111111111111111111"
	hash7 := HashTimelockOpId(calls1, differentPredecessor, salt)
	assert.NotEqual(t, hash1, hash7, "different predecessor should produce different hash")
}

func TestHashTimelockOpId_EmptyCalls(t *testing.T) {
	t.Parallel()

	predecessor := "0000000000000000000000000000000000000000000000000000000000000000"
	salt := "0000000000000000000000000000000000000000000000000000000000001234"

	// Empty calls should still produce a valid hash
	emptyCalls := []TimelockCall{}
	hash := HashTimelockOpId(emptyCalls, predecessor, salt)
	assert.NotEqual(t, common.Hash{}, hash, "empty calls should still produce a hash")

	// Hash with empty calls should be different from hash with calls
	nonEmptyCalls := []TimelockCall{
		{TargetInstanceId: "target", FunctionName: "func", OperationData: "1234"},
	}
	hashWithCalls := HashTimelockOpId(nonEmptyCalls, predecessor, salt)
	assert.NotEqual(t, hash, hashWithCalls, "empty calls should produce different hash than non-empty")
}

func TestHashTimelockOpId_MultipleCalls(t *testing.T) {
	t.Parallel()

	predecessor := "0000000000000000000000000000000000000000000000000000000000000000"
	salt := "0000000000000000000000000000000000000000000000000000000000001234"

	// Multiple calls
	calls := []TimelockCall{
		{TargetInstanceId: "target-1", FunctionName: "func1", OperationData: "1234"},
		{TargetInstanceId: "target-2", FunctionName: "func2", OperationData: "5678"},
	}
	hash := HashTimelockOpId(calls, predecessor, salt)
	assert.NotEqual(t, common.Hash{}, hash)

	// Order matters
	reversedCalls := []TimelockCall{
		{TargetInstanceId: "target-2", FunctionName: "func2", OperationData: "5678"},
		{TargetInstanceId: "target-1", FunctionName: "func1", OperationData: "1234"},
	}
	hashReversed := HashTimelockOpId(reversedCalls, predecessor, salt)
	assert.NotEqual(t, hash, hashReversed, "call order should affect hash")
}

func TestIsValidHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid hex lowercase", "abcd1234", true},
		{"valid hex uppercase", "ABCD1234", true},
		{"valid hex mixed case", "AbCd1234", true},
		{"valid empty string", "", true},
		{"invalid odd length", "abc", false},
		{"invalid characters", "ghij", false},
		{"invalid with spaces", "ab cd", false},
		{"valid zeros", "0000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isValidHex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeOperationData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid hex passed through", "abcd1234", "abcd1234"},
		{"ascii encoded to hex", "hello", "68656c6c6f"},
		{"odd length ascii encoded", "abc", "616263"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := encodeOperationData(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToMCMSTimelockCalls(t *testing.T) {
	t.Parallel()

	calls := []TimelockCall{
		{TargetInstanceId: "target-1", FunctionName: "func1", OperationData: "1234"},
		{TargetInstanceId: "target-2", FunctionName: "func2", OperationData: "5678"},
	}

	result := toMCMSTimelockCalls(calls)

	assert.Len(t, result, 2)
	assert.Equal(t, "target-1", string(result[0].TargetInstanceId))
	assert.Equal(t, "func1", string(result[0].FunctionName))
	assert.Equal(t, "1234", string(result[0].OperationData))
	assert.Equal(t, "target-2", string(result[1].TargetInstanceId))
	assert.Equal(t, "func2", string(result[1].FunctionName))
	assert.Equal(t, "5678", string(result[1].OperationData))
}
