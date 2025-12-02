package aptos

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/types"
)

func TestEncoderHashOperation(t *testing.T) {
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
				ChainSelector:        chaintest.Chain5Selector,
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
			want:    common.HexToHash("0x083b481c650e1578f7be562e21f23a570b4943fead4b1ffc67ac548166d8cf5c"),
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid Aptos chain selector",
			fields: fields{
				ChainSelector: chaintest.Chain1Selector,
			},
			wantErr: AssertErrorContains("not found for selector"),
		}, {
			name: "failure - invalid MCMAddress",
			fields: fields{
				ChainSelector:        chaintest.Chain5Selector,
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
				ChainSelector:        chaintest.Chain5Selector,
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
				ChainSelector:        chaintest.Chain5Selector,
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

func TestEncoderHashMetadata(t *testing.T) {
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
				ChainSelector:        chaintest.Chain5Selector,
				TxCount:              5,
				OverridePreviousRoot: false,
			},
			args: args{
				metadata: types.ChainMetadata{
					StartingOpCount: 7,
					MCMAddress:      "0x222",
				},
			},
			want:    common.HexToHash("0x455b5b8ec10c95fe066ca1736547d0758fa2a8356b39ca6aa9043ae1504fcc87"),
			wantErr: assert.NoError,
		}, {
			name: "success - with override previous root",
			fields: fields{
				ChainSelector:        chaintest.Chain5Selector,
				TxCount:              5,
				OverridePreviousRoot: true,
			},
			args: args{
				metadata: types.ChainMetadata{
					StartingOpCount: 7,
					MCMAddress:      "0x222",
				},
			},
			want:    common.HexToHash("0xe7b0cb38489c34616ce7af39d6d22afe4269778337e037a43a476c5179259911"),
			wantErr: assert.NoError,
		}, {
			name: "failure - invalid Aptos chain selector",
			fields: fields{
				ChainSelector: chaintest.Chain2Selector,
			},
			wantErr: AssertErrorContains("not found for selector"),
		}, {
			name: "failure - invalid MCMAddress",
			fields: fields{
				ChainSelector: chaintest.Chain5Selector,
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
