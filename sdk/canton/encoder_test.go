package canton

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/types"
)

func AssertErrorContains(errorMessage string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, msgAndArgs ...any) bool {
		if err == nil {
			return assert.Fail(t, "Expected error to be returned", msgAndArgs...)
		}
		return assert.Contains(t, err.Error(), errorMessage, msgAndArgs...)
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
				ChainSelector:        1,
				TxCount:              5,
				OverridePreviousRoot: false,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{
						"chainId": 1,
						"multisigId": "test-multisig",
						"preOpCount": 0,
						"postOpCount": 5,
						"overridePreviousRoot": false
					}`),
				},
				op: types.Operation{
					Transaction: types.Transaction{
						To:   "target-contract",
						Data: []byte{0x11, 0x22, 0x33, 0x44},
						AdditionalFields: json.RawMessage(`{
							"targetInstanceId": "instance-123",
							"functionName": "executeAction",
							"operationData": "1122334455",
							"targetCid": "cid-123"
						}`),
					},
				},
			},
			want:    common.HexToHash("0xe4f7155153e90245c12d484e518341394978318905a6710b230b54977170138a"),
			wantErr: assert.NoError,
		},
		{
			name: "success with different values",
			fields: fields{
				ChainSelector:        2,
				TxCount:              10,
				OverridePreviousRoot: true,
			},
			args: args{
				opCount: 5,
				metadata: types.ChainMetadata{
					MCMAddress: "11f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{
						"chainId": 123,
						"multisigId": "prod-multisig",
						"preOpCount": 5,
						"postOpCount": 15,
						"overridePreviousRoot": true
					}`),
				},
				op: types.Operation{
					Transaction: types.Transaction{
						To:   "another-target",
						Data: []byte{0xaa, 0xbb, 0xcc},
						AdditionalFields: json.RawMessage(`{
							"targetInstanceId": "prod-instance",
							"functionName": "transfer",
							"operationData": "aabbccdd",
							"targetCid": "cid-456"
						}`),
					},
				},
			},
			// Different encoding with different values
			want:    common.HexToHash("0xe391f3bfd945f06842d5dd286ecdabd1a148a94fe982e840b82db16daf6848ba"),
			wantErr: assert.NoError,
		},
		{
			name: "failure - invalid metadata additional fields JSON",
			fields: fields{
				ChainSelector: 1,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress:       "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{invalid json}`),
				},
				op: types.Operation{},
			},
			wantErr: AssertErrorContains("failed to unmarshal metadata additional fields"),
		},
		{
			name: "failure - invalid operation additional fields JSON",
			fields: fields{
				ChainSelector: 1,
			},
			args: args{
				opCount: 3,
				metadata: types.ChainMetadata{
					MCMAddress: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{
						"chainId": 1,
						"multisigId": "test-multisig",
						"preOpCount": 0,
						"postOpCount": 5
					}`),
				},
				op: types.Operation{
					Transaction: types.Transaction{
						AdditionalFields: json.RawMessage(`{invalid json}`),
					},
				},
			},
			wantErr: AssertErrorContains("failed to unmarshal operation additional fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := NewEncoder(tt.fields.ChainSelector, tt.fields.TxCount, tt.fields.OverridePreviousRoot)
			got, err := e.HashOperation(tt.args.opCount, tt.args.metadata, tt.args.op)

			tt.wantErr(t, err)
			if err == nil {
				t.Logf("Computed hash for %s: %s", tt.name, got.Hex())
				assert.Equal(t, tt.want, got, "hash should match expected value")
			}
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
			name: "success without override",
			fields: fields{
				ChainSelector:        1,
				TxCount:              5,
				OverridePreviousRoot: false,
			},
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{
						"chainId": 1,
						"multisigId": "test-multisig",
						"preOpCount": 0,
						"postOpCount": 5,
						"overridePreviousRoot": false
					}`),
				},
			},
			want:    common.HexToHash("0xb88e2ae0ecfa263c7a6fa6322e9aac55a06e722a1d2cf470cca361dd1325f9a2"),
			wantErr: assert.NoError,
		},
		{
			name: "success with override",
			fields: fields{
				ChainSelector:        2,
				TxCount:              10,
				OverridePreviousRoot: true,
			},
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "11f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{
						"chainId": 123,
						"multisigId": "prod-multisig",
						"preOpCount": 5,
						"postOpCount": 15,
						"overridePreviousRoot": true
					}`),
				},
			},
			want:    common.HexToHash("0x8d9da63765997aa703892ca624faaecf58c83d0201642ae313d98588a9705f08"),
			wantErr: assert.NoError,
		},
		{
			name: "success with different chain ID",
			fields: fields{
				ChainSelector: 3,
			},
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "22f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{
						"chainId": 999,
						"multisigId": "another-multisig",
						"preOpCount": 100,
						"postOpCount": 200,
						"overridePreviousRoot": false
					}`),
				},
			},
			want:    common.HexToHash("0x92794b1a017c4f55705839076c13ceb257e75fad47cb9420420e58cd38854351"),
			wantErr: assert.NoError,
		},
		{
			name: "failure - invalid metadata additional fields JSON",
			fields: fields{
				ChainSelector: 1,
			},
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress:       "00f8a3c8ed6c7e34bb3f3f16ed5d8a5fc9b7a6c1d2e3f4a5b6c7d8e9f0a1b2c3",
					AdditionalFields: json.RawMessage(`{invalid json}`),
				},
			},
			wantErr: AssertErrorContains("failed to unmarshal metadata additional fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := NewEncoder(tt.fields.ChainSelector, tt.fields.TxCount, tt.fields.OverridePreviousRoot)
			got, err := e.HashMetadata(tt.args.metadata)

			tt.wantErr(t, err)
			if err == nil {
				t.Logf("Computed hash for %s: %s", tt.name, got.Hex())
				assert.Equal(t, tt.want, got, "hash should match expected value")
			}
		})
	}
}

func TestPadLeft32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "pad short string",
			input: "1",
			want:  "0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			name:  "pad medium string",
			input: "123abc",
			want:  "000000000000000000000000000000000000000000000000000000000123abc",
		},
		{
			name:  "no padding needed",
			input: "0000000000000000000000000000000000000000000000000000000000000001",
			want:  "0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			name:  "truncate if too long",
			input: "00000000000000000000000000000000000000000000000000000000000000001",
			want:  "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:  "empty string",
			input: "",
			want:  "0000000000000000000000000000000000000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := padLeft32(tt.input)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, 64, len(got), "result should always be 64 characters")
		})
	}
}

func TestIntToHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int
		want  string
	}{
		{
			name:  "zero",
			input: 0,
			want:  "0",
		},
		{
			name:  "single digit",
			input: 5,
			want:  "5",
		},
		{
			name:  "two digits",
			input: 15,
			want:  "f",
		},
		{
			name:  "large number",
			input: 255,
			want:  "ff",
		},
		{
			name:  "very large number",
			input: 1234567,
			want:  "12d687",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := intToHex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAsciiToHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "hello",
			want:  "68656c6c6f",
		},
		{
			name:  "with hyphen",
			input: "test-multisig",
			want:  "746573742d6d756c7469736967",
		},
		{
			name:  "numbers",
			input: "12345",
			want:  "3132333435",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "special characters",
			input: "a@b.c",
			want:  "6140622e63",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := asciiToHex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewEncoder(t *testing.T) {
	t.Parallel()

	encoder := NewEncoder(123, 456, true)

	require.NotNil(t, encoder)
	assert.Equal(t, types.ChainSelector(123), encoder.ChainSelector)
	assert.Equal(t, uint64(456), encoder.TxCount)
	assert.True(t, encoder.OverridePreviousRoot)
}
