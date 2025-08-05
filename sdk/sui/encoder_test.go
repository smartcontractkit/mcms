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
