package sui

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"
)

func AssertErrorContains(errorMessage string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...any) bool {
		return assert.ErrorContains(t, err, errorMessage, i...)
	}
}
func TestEncoder_HashOperation(t *testing.T) {
	t.Parallel()
	type fields struct {
		ChainSelector        types.ChainSelector
		TxCount              uint64
		OverridePreviousRoot bool
	}
	type args struct {
		opCount  uint32
		metadata types.ChainMetadata
		op       types.Operation
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    common.Hash
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				ChainSelector:        chaintest.Chain6Selector,
				TxCount:              5,
				OverridePreviousRoot: true,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress: "0x111",
				},
				op: types.Operation{
					Transaction: types.Transaction{
						OperationMetadata: types.OperationMetadata{},
						To:                "0x222",
						Data:              []byte{0x11, 0x22, 0x33, 0x44},
						AdditionalFields:  json.RawMessage(`{"module_name":"module","function":"function_one"}`),
					},
				},
			},
			want:    common.HexToHash("0x273f491fddd57442ffe9cef7ba560a035bac034df0edf9e51b01b637c34babf9"),
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid Sui chain selector",
			fields: fields{
				ChainSelector: chaintest.Chain1Selector,
			},
			wantErr: AssertErrorContains("not found for selector"),
		}, {
			name: "failure - invalid MCMAddress",
			fields: fields{
				ChainSelector:        chaintest.Chain6Selector,
				TxCount:              5,
				OverridePreviousRoot: true,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress: "invalidaddress",
				},
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		}, {
			name: "failure - invalid to address",
			fields: fields{
				ChainSelector:        chaintest.Chain6Selector,
				TxCount:              5,
				OverridePreviousRoot: true,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress: "0x111",
				},
				op: types.Operation{
					Transaction: types.Transaction{
						OperationMetadata: types.OperationMetadata{},
						To:                "invalidaddress",
					},
				},
			},
			wantErr: AssertErrorContains("parse To address"),
		}, {
			name: "failure - invalid additionalFields",
			fields: fields{
				ChainSelector:        chaintest.Chain6Selector,
				TxCount:              5,
				OverridePreviousRoot: true,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress: "0x111",
				},
				op: types.Operation{
					Transaction: types.Transaction{
						OperationMetadata: types.OperationMetadata{},
						To:                "0x222",
						Data:              []byte{0x11, 0x22, 0x33, 0x44},
						AdditionalFields:  json.RawMessage(`{"module_name":"module",invalid json}`),
					},
				},
			},
			wantErr: AssertErrorContains("unmarshal additional fields"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEncoder(tt.fields.ChainSelector, tt.fields.TxCount, tt.fields.OverridePreviousRoot)
			got, err := e.HashOperation(tt.args.opCount, tt.args.metadata, tt.args.op)
			if !tt.wantErr(t, err, fmt.Sprintf("HashOperation(%v, %v, %v)", tt.args.opCount, tt.args.metadata, tt.args.op)) {
				return
			}
			assert.Equalf(t, tt.want.String(), got.String(), "HashOperation(%v, %v, %v)", tt.args.opCount, tt.args.metadata, tt.args.op)
		})
	}
}

func TestEncoder_HashMetadata(t *testing.T) {
	t.Parallel()
	type fields struct {
		ChainSelector        types.ChainSelector
		TxCount              uint64
		OverridePreviousRoot bool
	}
	type args struct {
		metadata types.ChainMetadata
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    common.Hash
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				ChainSelector:        chaintest.Chain6Selector,
				TxCount:              5,
				OverridePreviousRoot: false,
			},
			args: args{
				metadata: types.ChainMetadata{
					StartingOpCount: 7,
					MCMAddress:      "0x222",
				},
			},
			want:    common.HexToHash("0xed8b697fd845f8c1a493d1da979d91f0c4ab6acff6d0f86797029c3ecd8f8b57"),
			wantErr: assert.NoError,
		}, {
			name: "success - with override previous root",
			fields: fields{
				ChainSelector:        chaintest.Chain6Selector,
				TxCount:              5,
				OverridePreviousRoot: true,
			},
			args: args{
				metadata: types.ChainMetadata{
					StartingOpCount: 7,
					MCMAddress:      "0x222",
				},
			},
			want:    common.HexToHash("0xf8345dcc86304d6b38a38976b81bbc05ee8b194d44d6c5fb3564b110d0d25d1d"),
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid Sui chain selector",
			fields: fields{
				ChainSelector: chaintest.Chain2Selector,
			},
			wantErr: AssertErrorContains("not found for selector"),
		}, {
			name: "failure - invalid MCMAddress",
			fields: fields{
				ChainSelector: chaintest.Chain6Selector,
				TxCount:       5,
			},
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "invalidaddress!",
				},
			},
			wantErr: AssertErrorContains("parse MCMS address"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEncoder(tt.fields.ChainSelector, tt.fields.TxCount, tt.fields.OverridePreviousRoot)
			got, err := e.HashMetadata(tt.args.metadata)
			if !tt.wantErr(t, err, fmt.Sprintf("HashMetadata(%v)", tt.args.metadata)) {
				return
			}
			assert.Equalf(t, tt.want.String(), got.String(), "HashMetadata(%v)", tt.args.metadata)
		})
	}
}

func TestSerializeTimelockBypasserExecuteBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		stateObjects  []string
		targets       [][]byte
		moduleNames   []string
		functionNames []string
		datas         [][]byte
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			name:          "success - single call",
			stateObjects:  []string{"state_obj_1"},
			targets:       [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}},
			moduleNames:   []string{"test_module"},
			functionNames: []string{"test_function"},
			datas:         [][]byte{{0xaa, 0xbb, 0xcc}},
			wantErr:       assert.NoError,
		},
		{
			name:         "success - multiple calls",
			stateObjects: []string{"state_obj_1", "state_obj_2"},
			targets: [][]byte{
				{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
				{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
			},
			moduleNames:   []string{"module_1", "module_2"},
			functionNames: []string{"function_1", "function_2"},
			datas:         [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd, 0xee}},
			wantErr:       assert.NoError,
		},
		{
			name:          "success - empty state objects",
			stateObjects:  []string{},
			targets:       [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}},
			moduleNames:   []string{"test_module"},
			functionNames: []string{"test_function"},
			datas:         [][]byte{{0xaa, 0xbb, 0xcc}},
			wantErr:       assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := SerializeTimelockBypasserExecuteBatch(tt.stateObjects, tt.targets, tt.moduleNames, tt.functionNames, tt.datas)
			if !tt.wantErr(t, err, "SerializeTimelockBypasserExecuteBatch should not error") {
				return
			}

			// Verify the data can be deserialized back
			if err == nil {
				calls, deserializeErr := DeserializeTimelockBypasserExecuteBatch(data)
				assert.NoError(t, deserializeErr, "Should be able to deserialize the serialized data")
				assert.Equal(t, len(tt.targets), len(calls), "Number of calls should match number of targets")

				for i, call := range calls {
					assert.Equal(t, tt.targets[i], call.Target, "Target should match")
					assert.Equal(t, tt.moduleNames[i], call.ModuleName, "Module name should match")
					assert.Equal(t, tt.functionNames[i], call.FunctionName, "Function name should match")
					assert.Equal(t, tt.datas[i], call.Data, "Data should match")

					// Check state object assignment
					if len(tt.stateObjects) == len(tt.targets) {
						assert.Equal(t, tt.stateObjects[i], call.StateObj, "State object should match")
					} else {
						assert.Equal(t, "", call.StateObj, "State object should be empty when count doesn't match")
					}
				}
			}
		})
	}
}

func TestRemoveStateObjectsFromBypassData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		stateObjects  []string
		targets       [][]byte
		moduleNames   []string
		functionNames []string
		datas         [][]byte
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			name:          "success - single call with state object",
			stateObjects:  []string{"state_obj_1"},
			targets:       [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}},
			moduleNames:   []string{"test_module"},
			functionNames: []string{"test_function"},
			datas:         [][]byte{{0xaa, 0xbb, 0xcc}},
			wantErr:       assert.NoError,
		},
		{
			name:         "success - multiple calls with state objects",
			stateObjects: []string{"state_obj_1", "state_obj_2"},
			targets: [][]byte{
				{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
				{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
			},
			moduleNames:   []string{"module_1", "module_2"},
			functionNames: []string{"function_1", "function_2"},
			datas:         [][]byte{{0xaa, 0xbb}, {0xcc, 0xdd, 0xee}},
			wantErr:       assert.NoError,
		},
		{
			name:          "success - empty state objects",
			stateObjects:  []string{},
			targets:       [][]byte{{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}},
			moduleNames:   []string{"test_module"},
			functionNames: []string{"test_function"},
			datas:         [][]byte{{0xaa, 0xbb, 0xcc}},
			wantErr:       assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// First serialize with state objects
			originalData, err := SerializeTimelockBypasserExecuteBatch(tt.stateObjects, tt.targets, tt.moduleNames, tt.functionNames, tt.datas)
			assert.NoError(t, err, "Serialization should not fail")

			// Remove state objects
			dataWithoutStateObjects, err := RemoveStateObjectsFromBypassData(originalData)
			if !tt.wantErr(t, err, "RemoveStateObjectsFromBypassData should not error") {
				return
			}

			if err == nil {
				// Verify the result can be deserialized (but state objects should be empty)
				calls, deserializeErr := DeserializeTimelockBypasserExecuteBatchWithoutStateObjects(dataWithoutStateObjects)
				assert.NoError(t, deserializeErr, "Should be able to deserialize data without state objects")
				assert.Equal(t, len(tt.targets), len(calls), "Number of calls should match number of targets")

				for i, call := range calls {
					assert.Equal(t, tt.targets[i], call.Target, "Target should match")
					assert.Equal(t, tt.moduleNames[i], call.ModuleName, "Module name should match")
					assert.Equal(t, tt.functionNames[i], call.FunctionName, "Function name should match")
					assert.Equal(t, tt.datas[i], call.Data, "Data should match")
					assert.Equal(t, "", call.StateObj, "State object should be empty after removal")
				}

				// Verify the data without state objects is different from original (unless there were no state objects)
				if len(tt.stateObjects) > 0 {
					assert.NotEqual(t, originalData, dataWithoutStateObjects, "Data should be different after removing state objects")
				} else {
					// When there are no state objects, the RemoveStateObjectsFromBypassData function should
					// still remove the empty vector prefix (0x00), so the data should be different
					assert.NotEqual(t, originalData, dataWithoutStateObjects, "Data should be different after removing empty state objects vector")
				}
			}
		})
	}
}
