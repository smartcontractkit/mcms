package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	solana "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
)

func TestNewExecutor(t *testing.T) {
	type args struct {
		client  *rpc.Client
		auth    solana.PrivateKey
		encoder *Encoder
	}
	tests := []struct {
		name string
		args args
		want *Executor
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, NewExecutor(tt.args.client, tt.args.auth, tt.args.encoder), "%q. NewExecutor()", tt.name)
	}
}

func TestExecutor_ExecuteOperation(t *testing.T) {
	type fields struct {
		Encoder   *Encoder
		Inspector *Inspector
		client    *rpc.Client
		auth      solana.PrivateKey
	}
	type args struct {
		metadata types.ChainMetadata
		nonce    uint32
		proof    []common.Hash
		op       types.Operation
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		e := &Executor{
			Encoder:   tt.fields.Encoder,
			Inspector: tt.fields.Inspector,
			client:    tt.fields.client,
			auth:      tt.fields.auth,
		}
		got, err := e.ExecuteOperation(tt.args.metadata, tt.args.nonce, tt.args.proof, tt.args.op)
		tt.assertion(t, err, fmt.Sprintf("%q. Executor.ExecuteOperation()", tt.name))
		assert.Equalf(t, tt.want, got, "%q. Executor.ExecuteOperation()", tt.name)
	}
}

func TestExecutor_SetRoot(t *testing.T) {
	type fields struct {
		Encoder   *Encoder
		Inspector *Inspector
		client    *rpc.Client
		auth      solana.PrivateKey
	}
	type args struct {
		metadata         types.ChainMetadata
		proof            []common.Hash
		root             [32]byte
		validUntil       uint32
		sortedSignatures []types.Signature
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      string
		assertion assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		e := &Executor{
			Encoder:   tt.fields.Encoder,
			Inspector: tt.fields.Inspector,
			client:    tt.fields.client,
			auth:      tt.fields.auth,
		}
		got, err := e.SetRoot(tt.args.metadata, tt.args.proof, tt.args.root, tt.args.validUntil, tt.args.sortedSignatures)
		tt.assertion(t, err, fmt.Sprintf("%q. Executor.SetRoot()", tt.name))
		assert.Equalf(t, tt.want, got, "%q. Executor.SetRoot()", tt.name)
	}
}

func TestExecutor_preloadSignatures(t *testing.T) {
	type fields struct {
		Encoder   *Encoder
		Inspector *Inspector
		client    *rpc.Client
		auth      solana.PrivateKey
	}
	type args struct {
		ctx              context.Context
		mcmName          [32]byte
		root             [32]byte
		validUntil       uint32
		sortedSignatures []types.Signature
		signaturesPDA    solana.PublicKey
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		assertion assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		e := &Executor{
			Encoder:   tt.fields.Encoder,
			Inspector: tt.fields.Inspector,
			client:    tt.fields.client,
			auth:      tt.fields.auth,
		}
		tt.assertion(t, e.preloadSignatures(tt.args.ctx, tt.args.mcmName, tt.args.root, tt.args.validUntil, tt.args.sortedSignatures, tt.args.signaturesPDA), fmt.Sprintf("%q. Executor.preloadSignatures()", tt.name))
	}
}

func TestExecutor_solanaMetadata(t *testing.T) {
	type fields struct {
		Encoder   *Encoder
		Inspector *Inspector
		client    *rpc.Client
		auth      solana.PrivateKey
	}
	type args struct {
		metadata      types.ChainMetadata
		configPDA     [32]byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   mcm.RootMetadataInput
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		e := &Executor{
			Encoder:   tt.fields.Encoder,
			Inspector: tt.fields.Inspector,
			client:    tt.fields.client,
			auth:      tt.fields.auth,
		}
		assert.Equalf(t, tt.want, e.solanaMetadata(tt.args.metadata, tt.args.configPDA), "%q. Executor.solanaMetadata()", tt.name)
	}
}

func Test_solanaProof(t *testing.T) {
	type args struct {
		proof []common.Hash
	}
	tests := []struct {
		name string
		args args
		want [][32]uint8
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, solanaProof(tt.args.proof), "%q. solanaProof()", tt.name)
	}
}

func Test_solanaSignatures(t *testing.T) {
	type args struct {
		signatures []types.Signature
	}
	tests := []struct {
		name string
		args args
		want []mcm.Signature
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, solanaSignatures(tt.args.signatures), "%q. solanaSignatures()", tt.name)
	}
}
