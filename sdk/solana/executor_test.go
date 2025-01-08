package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

var McmProgram = solana.MustPublicKeyFromBase58("6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX")
var McmName = "test-mcms"

const dummyPrivateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

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

	type args struct {
		metadata types.ChainMetadata
		nonce    uint32
		proof    []common.Hash
		op       types.Operation
	}
	selector := cselectors.SOLANA_DEVNET.Selector
	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)
	contractID := fmt.Sprintf("%s.%s", McmProgram.String(), McmName)
	data := []byte{1, 2, 3, 4}
	accounts := []solana.AccountMeta{}
	tx := NewTransaction(McmProgram.String(), data, accounts, "solana-testing", []string{})
	tests := []struct {
		name string
		args args

		mockSetup func(*mocks.JSONRPCClient)
		want      string
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{
			name: "Test ExecuteOperation",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: contractID,
				},
				nonce: 0,
				proof: []common.Hash{
					common.HexToHash("0x1"),
				},
				op: types.Operation{
					Transaction:   tx,
					ChainSelector: types.ChainSelector(selector),
				},
			},
			mockSetup: func(m *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, m, 20, 5, "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", nil)
			},
			want:      "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
			assertion: assert.NoError,
		},
	}
	for _, tt := range tests {
		jsonRPCClient := mocks.NewJSONRPCClient(t)
		encoder := NewEncoder(types.ChainSelector(selector), uint64(tt.args.nonce), false)
		client := rpc.NewWithCustomRPCClient(jsonRPCClient)
		tt.mockSetup(jsonRPCClient)
		e := NewExecutor(client, auth, encoder)

		got, err := e.ExecuteOperation(tt.args.metadata, tt.args.nonce, tt.args.proof, tt.args.op)
		if tt.wantErr != nil {
			assert.ErrorIs(t, err, tt.wantErr)
		} else {
			assert.NoError(t, err)
		}
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
		metadata  types.ChainMetadata
		configPDA [32]byte
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
