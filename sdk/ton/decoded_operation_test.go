package ton

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDecodedOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		contractType string
		msgType      string
		msgOpcode    uint64
		msgDecoded   map[string]interface{}
		inputKeys    []string
		inputArgs    []any
		wantMethod   string
		wantErr      string
	}{
		{
			name:         "success",
			contractType: "com.foo.acc",
			msgType:      "functionName",
			msgOpcode:    0x1,
			msgDecoded: map[string]interface{}{
				"inputKey1": "inputArg1",
				"inputKey2": "inputArg2",
			},
			inputKeys:  []string{"inputKey1", "inputKey2"},
			inputArgs:  []any{"inputArg1", "inputArg2"},
			wantMethod: "com.foo.acc::functionName(0x1)",
		},
		{
			name:         "success with empty input keys and args",
			contractType: "com.foo.acc",
			msgType:      "functionName",
			msgOpcode:    0x7362d09c,
			msgDecoded:   map[string]interface{}{},
			inputKeys:    []string{},
			inputArgs:    []any{},
			wantMethod:   "com.foo.acc::functionName(0x7362d09c)",
		},
		{
			name:         "error with mismatched input keys and args",
			contractType: "com.foo.acc",
			msgType:      "functionName",
			msgOpcode:    0x1,
			msgDecoded: map[string]interface{}{
				"inputKey1": "inputArg1",
				"inputKey2": "inputArg2",
			},
			inputKeys:  []string{"inputKey1", "inputKey2"},
			inputArgs:  []any{"inputArg1"},
			wantMethod: "com.foo.acc::functionName(0x1)",
			wantErr:    "input keys and input args must have the same length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewDecodedOperation(tt.contractType, tt.msgType, tt.msgOpcode, tt.msgDecoded, tt.inputKeys, tt.inputArgs)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)

				// Test member functions
				assert.Equal(t, tt.wantMethod, got.MethodName())
				assert.Equal(t, tt.inputKeys, got.Keys())
				assert.Equal(t, tt.inputArgs, got.Args())

				// Test String()
				fn, inputs, err := got.String()
				require.NoError(t, err)
				assert.Equal(t, tt.wantMethod, fn)
				for i := range tt.inputKeys {
					assert.Contains(t, inputs, fmt.Sprintf(`"%s": "%v"`, tt.inputKeys[i], tt.inputArgs[i]))
				}
			}
		})
	}
}
