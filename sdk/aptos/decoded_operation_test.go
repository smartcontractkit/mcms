package aptos

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
		packageName  string
		moduleName   string
		functionName string
		inputKeys    []string
		inputArgs    []any
		wantErr      string
	}{
		{
			name:         "success",
			packageName:  "package_name",
			moduleName:   "module_name",
			functionName: "function_one",
			inputKeys:    []string{"inputKey1", "inputKey2"},
			inputArgs:    []any{"inputArg1", "inputArg2"},
		},
		{
			name:         "success with empty input keys and args",
			packageName:  "package_name",
			moduleName:   "module_name",
			functionName: "function_two",
			inputKeys:    []string{},
			inputArgs:    []any{},
		},
		{
			name:         "error with mismatched input keys and args",
			packageName:  "package_name",
			moduleName:   "module_name",
			functionName: "function_three",
			inputKeys:    []string{"inputKey1", "inputKey2"},
			inputArgs:    []any{"inputArg1"},
			wantErr:      "input keys and input args must have the same length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewDecodedOperation(tt.packageName, tt.moduleName, tt.functionName, tt.inputKeys, tt.inputArgs)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.functionName, got.FunctionName)
				assert.Equal(t, tt.inputKeys, got.InputKeys)
				assert.Equal(t, tt.inputArgs, got.InputArgs)

				// Test member functions
				assert.Equal(t, fmt.Sprintf("%s::%s::%s", tt.packageName, tt.moduleName, tt.functionName), got.MethodName())
				assert.Equal(t, tt.inputKeys, got.Keys())
				assert.Equal(t, tt.inputArgs, got.Args())

				// Test String()
				fn, inputs, err := got.String()
				require.NoError(t, err)
				assert.Equal(t, tt.functionName, fn)
				for i := range tt.inputKeys {
					assert.Contains(t, inputs, fmt.Sprintf(`"%s": "%v"`, tt.inputKeys[i], tt.inputArgs[i]))
				}
			}
		})
	}
}
