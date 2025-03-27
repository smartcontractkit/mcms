package aptos

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertErrorContains(errorMessage string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...any) bool {
		return assert.ErrorContains(t, err, errorMessage, i...)
	}
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}

func TestArgsToData(t *testing.T) {
	t.Parallel()
	got := ArgsToData([][]byte{
		{0x1, 0x2},
		{0x3, 0x4},
	})
	require.Equal(t, []byte{0x1, 0x2, 0x3, 0x4}, got)
}

func TestValidateAdditionalFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		additionalFields json.RawMessage
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name:             "success",
			additionalFields: []byte(`{"package_name":"package","module_name":"module","function":"function"}`),
			wantErr:          assert.NoError,
		}, {
			name:             "failure - invalid json",
			additionalFields: []byte(`{"invalidjson","package_name":"package","module_name":"module","function":"function"}`),
			wantErr:          AssertErrorContains("unmarshal"),
		}, {
			name:             "failure - missing package",
			additionalFields: []byte(`{"module_name":"module","function":"function"}`),
			wantErr:          AssertErrorContains("package name"),
		}, {
			name:             "failure - missing module",
			additionalFields: []byte(`{"package_name":"package","function":"function"}`),
			wantErr:          AssertErrorContains("module name"),
		}, {
			name:             "failure - invalid module",
			additionalFields: []byte(`{"package_name":"package","module_name":"thismodulenameiswithoutadoubtwaytoolongasitsnamehasseventyonecharacters","function":"function"}`),
			wantErr:          AssertErrorContains("module name"),
		}, {
			name:             "failure - missing function",
			additionalFields: []byte(`{"package_name":"package","module_name":"module"}`),
			wantErr:          AssertErrorContains("function"),
		}, {
			name:             "failure - invalid function",
			additionalFields: []byte(`{"package_name":"package","module_name":"module","function":"thisfunctionnameisdefinitelywaytolongasitislongerthansixtyfourcharacters"}`),
			wantErr:          AssertErrorContains("function"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.wantErr(t, ValidateAdditionalFields(tt.additionalFields), fmt.Sprintf("ValidateAdditionalFields(%v)", tt.additionalFields))
		})
	}
}
