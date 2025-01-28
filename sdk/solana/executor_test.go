package solana

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

const dummyPrivateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth := solana.MustPrivateKeyFromBase58("DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc")
	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)
	encoder := NewEncoder(chainSelector, 1, false)

	executor := NewExecutor(client, auth, encoder)

	require.NotNil(t, executor)
}

func TestExecutor_ExecuteOperation(t *testing.T) {
	t.Parallel()

	type args struct {
		metadata types.ChainMetadata
		nonce    uint32
		proof    []common.Hash
		op       types.Operation
	}
	selector := cselectors.SOLANA_DEVNET.Selector
	auth, err := solana.PrivateKeyFromBase58(dummyPrivateKey)
	require.NoError(t, err)
	contractID := fmt.Sprintf("%s.%s", testMCMProgramID.String(), testPDASeed)
	data := []byte{1, 2, 3, 4}
	ctx := context.Background()
	accounts := []*solana.AccountMeta{}
	tx, err := NewTransaction(testMCMProgramID.String(), data, big.NewInt(0), accounts, "solana-testing", []string{})
	require.NoError(t, err)
	tests := []struct {
		name string
		args args

		mockSetup func(*mocks.JSONRPCClient)
		want      string
		assertion assert.ErrorAssertionFunc
		wantErr   error
	}{
		{
			name: "success: ExecuteOperation",
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
		{
			name: "error: invalid contract ID provided",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: "bad id",
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

			},
			want:    "",
			wantErr: errors.New("invalid contract ID provided"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "invalid solana contract address format: \"bad id\"")
			},
		},
		{
			name: "error: ix send failure",
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
				mockSolanaTransaction(t,
					m,
					20,
					5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					errors.New("ix send failure"))
			},
			want:    "",
			wantErr: errors.New("invalid contract ID provided"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to call execute operation instruction: unable to send instruction: ix send failure")
			},
		},
		{
			name: "error: invalid additional fields",
			args: args{
				metadata: types.ChainMetadata{
					MCMAddress: contractID,
				},
				nonce: 0,
				proof: []common.Hash{
					common.HexToHash("0x1"),
				},
				op: types.Operation{
					Transaction: types.Transaction{
						AdditionalFields: []byte("bad data"),
					},
					ChainSelector: types.ChainSelector(selector),
				},
			},
			mockSetup: func(m *mocks.JSONRPCClient) {

			},
			want:    "",
			wantErr: errors.New("invalid contract ID provided"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to unmarshal additional fields: invalid character 'b' looking for beginning of value")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jsonRPCClient := mocks.NewJSONRPCClient(t)
			encoder := NewEncoder(types.ChainSelector(selector), uint64(tt.args.nonce), false)
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			tt.mockSetup(jsonRPCClient)
			e := NewExecutor(client, auth, encoder)

			got, err := e.ExecuteOperation(ctx, tt.args.metadata, tt.args.nonce, tt.args.proof, tt.args.op)
			if tt.wantErr != nil {
				tt.assertion(t, err, fmt.Sprintf("%q. Executor.ExecuteOperation()", tt.name))
			} else {
				require.NoError(t, err)
				require.Equalf(t, tt.want, got, "%q. Executor.ExecuteOperation()", tt.name)
			}
		})
	}
}

func TestExecutor_SetRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	defaultMetadata := types.ChainMetadata{StartingOpCount: 100, MCMAddress: ContractAddress(testMCMProgramID, testPDASeed)}
	defaultProof := []common.Hash{common.HexToHash("0x1"), common.HexToHash("0x2")}
	defaultRoot := common.HexToHash("0x1234")
	defaultValidUntil := uint32(2082758400)
	defaultSignatures := []types.Signature{
		{R: common.HexToHash("0x3"), S: common.HexToHash("0x4"), V: 27},
		{R: common.HexToHash("0x5"), S: common.HexToHash("0x6"), V: 27},
	}

	tests := []struct {
		name       string
		metadata   types.ChainMetadata
		proof      []common.Hash
		root       [32]byte
		validUntil uint32
		signatures []types.Signature
		setup      func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient)
		want       string
		wantErr    string
	}{
		{
			name:       "success",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// TODO: extract/decode payload in transaction data and test values
				// 4 transactions: init-signatures, append-signatures, finalize-signatures, set-root
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 51, 61,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 52, 62,
					"GHR9z23oUJnS2aV5HZ9zEpUpEe4qoBLFMzuBjNA3xJcQuc6JjDmuUq2VVmxqwPeFzfs8V7nfjqc1wRviEb82bRu", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 53, 63,
					"oaV9FKKPDVneUANQ9hJqEuhgwfUgbxucUC4TmzpgGJhuSxBueapWc9HJ4cJQMqT2PPQX6rhTbKnXkebsaravnLo", nil)
			},
			want: "oaV9FKKPDVneUANQ9hJqEuhgwfUgbxucUC4TmzpgGJhuSxBueapWc9HJ4cJQMqT2PPQX6rhTbKnXkebsaravnLo",
		},
		{
			name:       "failure: invalid address",
			metadata:   types.ChainMetadata{StartingOpCount: 100, MCMAddress: "invalid-mcm-address"},
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup:      func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) { t.Helper() },
			wantErr:    "invalid solana contract address format: \"invalid-mcm-address\"",
		},
		{
			name:       "failure: too many signatures",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: generateSignatures(t, 256),
			setup:      func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) { t.Helper() },
			wantErr:    "too many signatures (max 255)",
		},
		{
			name:       "failure: initialize signatures error",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// init-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v",
					fmt.Errorf("initialize signatures error"))
			},
			wantErr: "unable to initialize signatures: unable to send instruction: initialize signatures error",
		},
		{
			name:       "failure: append signatures error",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// init-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil)

				// append-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr",
					fmt.Errorf("append signatures error"))
			},
			wantErr: "unable to append signatures (0): unable to send instruction: append signatures error",
		},
		{
			name:       "failure: finalize signatures error",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// init-signatures + append-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr", nil)

				// finalize-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 52, 62,
					"GHR9z23oUJnS2aV5HZ9zEpUpEe4qoBLFMzuBjNA3xJcQuc6JjDmuUq2VVmxqwPeFzfs8V7nfjqc1wRviEb82bRu",
					fmt.Errorf("finalize signatures error"))
			},
			wantErr: "unable to finalize signatures: unable to send instruction: finalize signatures error",
		},
		{
			name:       "failure: set-root error",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// init-signatures + append-signatures + finalize-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 52, 62,
					"GHR9z23oUJnS2aV5HZ9zEpUpEe4qoBLFMzuBjNA3xJcQuc6JjDmuUq2VVmxqwPeFzfs8V7nfjqc1wRviEb82bRu", nil)

				// set-root
				mockSolanaTransaction(t, mockJSONRPCClient, 53, 63,
					"oaV9FKKPDVneUANQ9hJqEuhgwfUgbxucUC4TmzpgGJhuSxBueapWc9HJ4cJQMqT2PPQX6rhTbKnXkebsaravnLo",
					fmt.Errorf("set root error"))
			},
			wantErr: "unable to set root: unable to send instruction: set root error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor, mockJSONRPCClient := newTestExecutor(t, auth, chainSelector)
			tt.setup(t, executor, mockJSONRPCClient)

			got, err := executor.SetRoot(ctx, tt.metadata, tt.proof, tt.root, tt.validUntil, tt.signatures)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

func newTestExecutor(t *testing.T, auth solana.PrivateKey, chainSelector types.ChainSelector) (*Executor, *mocks.JSONRPCClient) {
	t.Helper()

	mockJSONRPCClient := mocks.NewJSONRPCClient(t)
	client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)
	encoder := NewEncoder(chainSelector, 1, false)

	return NewExecutor(client, auth, encoder), mockJSONRPCClient
}
