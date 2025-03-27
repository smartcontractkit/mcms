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
			want:    common.HexToHash("0xae71ad695a3f8ca57cc0e6c0ac470f697bf572462c41e76a0525775541d1e31d"),
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
			want:    common.HexToHash("0x473abb0ff4139883be8748f0707b1f3aad4861f5e8382521f0d54a446f2fb29e"),
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
			want:    common.HexToHash("0x063fb778d54011efa7481bb914e45654b77f2aef77a2eb698107042b44b1a6d5"),
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
