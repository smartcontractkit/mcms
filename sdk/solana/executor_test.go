package solana

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/mcm"

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

	executor := NewExecutor(encoder, client, auth)

	require.NotNil(t, executor)
}

func TestExecutorExecuteOperation(t *testing.T) { //nolint:paralleltest
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
		name      string
		args      args
		setup     func(*testing.T, *mocks.JSONRPCClient)
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
			setup: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				mockSolanaTransaction(t, m, 20, 5,
					"2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6", nil, nil)
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
			setup:   func(t *testing.T, m *mocks.JSONRPCClient) { t.Helper() },
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
			setup: func(t *testing.T, m *mocks.JSONRPCClient) {
				t.Helper()
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockSolanaTransaction(t, m, 20, 5, "2QUBE2GqS8PxnGP1EBrWpLw3La4XkEUz5NKXJTdTHoA43ANkf5fqKwZ8YPJVAi3ApefbbbCYJipMVzUa7kg3a7v6",
					nil, errors.New("ix send failure"))
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
			setup:   func(t *testing.T, m *mocks.JSONRPCClient) { t.Helper() },
			want:    "",
			wantErr: errors.New("invalid contract ID provided"),
			assertion: func(t assert.TestingT, err error, i ...any) bool {
				return assert.EqualError(t, err, "unable to unmarshal additional fields: invalid character 'b' looking for beginning of value")
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			jsonRPCClient := mocks.NewJSONRPCClient(t)
			encoder := NewEncoder(types.ChainSelector(selector), uint64(tt.args.nonce), false)
			client := rpc.NewWithCustomRPCClient(jsonRPCClient)
			executor := NewExecutor(encoder, client, auth)

			tt.setup(t, jsonRPCClient)

			got, err := executor.ExecuteOperation(ctx, tt.args.metadata, tt.args.nonce, tt.args.proof, tt.args.op)
			if tt.wantErr != nil {
				tt.assertion(t, err, fmt.Sprintf("%q. Executor.ExecuteOperation()", tt.name))
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.NotNil(t, got.RawData)
				require.Equalf(t, tt.want, got.Hash, "%q. Executor.ExecuteOperation()", tt.name)
			}
		})
	}
}

func TestExecutorSetRoot(t *testing.T) { //nolint:paralleltest
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
	defaultPreviousRoot := common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdef")
	defaultRootAndOpCount := &bindings.ExpiringRootAndOpCount{Root: defaultPreviousRoot, ValidUntil: 123}
	opCountPDA, err := FindExpiringRootAndOpCountPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)

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
			name:       "success - direct",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)

				// TODO: extract/decode payload in transaction data and test values
				// 4 transactions: init-signatures, append-signatures, finalize-signatures, set-root
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 51, 61,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 52, 62,
					"GHR9z23oUJnS2aV5HZ9zEpUpEe4qoBLFMzuBjNA3xJcQuc6JjDmuUq2VVmxqwPeFzfs8V7nfjqc1wRviEb82bRu", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 53, 63,
					"oaV9FKKPDVneUANQ9hJqEuhgwfUgbxucUC4TmzpgGJhuSxBueapWc9HJ4cJQMqT2PPQX6rhTbKnXkebsaravnLo", nil, nil)
			},
			want: "oaV9FKKPDVneUANQ9hJqEuhgwfUgbxucUC4TmzpgGJhuSxBueapWc9HJ4cJQMqT2PPQX6rhTbKnXkebsaravnLo",
		},
		{
			name:       "success - after account already in use error",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: defaultSignatures,
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)

				accountAlreadyInUseError := errors.New(`
					(string) (len=4) "logs": ([]interface {}) (len=7 cap=8) {
					(string) (len=63) "Program 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk invoke [1]",
					(string) (len=40) "Program log: Instruction: InitSignatures",
					(string) (len=51) "Program 11111111111111111111111111111111 invoke [2]",
					(string) (len=110) "Allocate: account Address { address: DKvwoHhcRMZwsedUvJqmsNuat52WdcYQD7agqMjJybbF, base: None } already in use",
					(string) (len=74) "Program 11111111111111111111111111111111 failed: custom program error: 0x0",
					(string) (len=90) "Program 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk consumed 7202 of 200000 compute units",
					(string) (len=86) "Program 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk failed: custom program error: 0x0"`)

				// 6 transactions: init-signatures, clear-signatures, init-signatures append-signatures, finalize-signatures, set-root
				mockSolanaTransaction(t, mockJSONRPCClient, 80, 70,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil, accountAlreadyInUseError)
				mockSolanaTransaction(t, mockJSONRPCClient, 81, 71,
					"5qm3BUCF1DswRm4r32mipWZa5NrbHYgPst8BJXr1BysNaqfEz4kVGnGzCx3vLWJoWi2FRszzgLUxfcmAfLzzHw9n", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 82, 72,
					"KCAkcwG8LG3cS3bUemdR1grv6EoyqfgDJN3BJfN58azEDkpRFa9S66RDFvtNWHga9htimSnfNkGoWVLLx7AJKrs", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 83, 63,
					"4GWCdxQAsAmbqyaB1SCHpmnyRWmBwBY5S6hUpSMz2gENErgt8zqmsXnz9dbXCchucZqAf7ZdctzTNtUUTjD8rMcv", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 84, 64,
					"npfXXzfJzSkB6QcwS36P9dzc61itiDnLX9yGVyzaooftiSFs1A53JWHGQq6F9MzjWxKD8bRLPKWuGseUxHK56s3", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 85, 65,
					"2W1d6qQAVPjssJgm6WMeLxwatKZ9TUUZcvNHdFaguvLxrCFqPdpuikiz9YdfkXG5eLojQNjrW6L6W2sFRWEujER", nil, nil)
			},
			want: "2W1d6qQAVPjssJgm6WMeLxwatKZ9TUUZcvNHdFaguvLxrCFqPdpuikiz9YdfkXG5eLojQNjrW6L6W2sFRWEujER",
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
			name:       "failure: unable to GetRoot",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: generateSignatures(t, 256),
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, errors.New("error"))
			},
			wantErr: "failed to get root: error",
		},
		{
			name:       "failure: GetRoot returns the same root",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: generateSignatures(t, 256),
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				rootAndOpCount := &bindings.ExpiringRootAndOpCount{Root: defaultRoot, ValidUntil: 123}
				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, rootAndOpCount, nil)
			},
			wantErr: "SignedHashAlreadySeen: 0x0000000000000000000000000000000000000000000000000000000000001234",
		},
		{
			name:       "failure: too many signatures",
			metadata:   defaultMetadata,
			proof:      defaultProof,
			root:       defaultRoot,
			validUntil: defaultValidUntil,
			signatures: generateSignatures(t, 256),
			setup: func(t *testing.T, executor *Executor, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()
				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)
			},
			wantErr: "too many signatures (max 255)",
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
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)

				// init-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v",
					nil, errors.New("initialize signatures error"))
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
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)

				// init-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil, nil)

				// append-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr",
					nil, errors.New("append signatures error"))
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
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)

				// init-signatures + append-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr", nil, nil)

				// finalize-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 52, 62,
					"GHR9z23oUJnS2aV5HZ9zEpUpEe4qoBLFMzuBjNA3xJcQuc6JjDmuUq2VVmxqwPeFzfs8V7nfjqc1wRviEb82bRu",
					nil, errors.New("finalize signatures error"))
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
				t.Setenv("MCMS_SOLANA_MAX_RETRIES", "1")

				mockGetAccountInfo(t, mockJSONRPCClient, opCountPDA, defaultRootAndOpCount, nil)

				// init-signatures + append-signatures + finalize-signatures
				mockSolanaTransaction(t, mockJSONRPCClient, 50, 60,
					"AxzwxQ2DLR4zEFxEPGaafR4z3MY4CP1CAdSs1ZZhArtgS3G4F9oYSy3Nx1HyA1Macb4bYEi4jU6F1CL4SRrZz1v", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"3DqeyZzb7PJmQ31M1qdXfP7iACr5AiEXcKeLUNmVvDoYM23JJK5ZvermxsDy8eiQKzpagc69MKRtrpzK7tRcLGgr", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 52, 62,
					"GHR9z23oUJnS2aV5HZ9zEpUpEe4qoBLFMzuBjNA3xJcQuc6JjDmuUq2VVmxqwPeFzfs8V7nfjqc1wRviEb82bRu", nil, nil)

				// set-root
				mockSolanaTransaction(t, mockJSONRPCClient, 53, 63,
					"oaV9FKKPDVneUANQ9hJqEuhgwfUgbxucUC4TmzpgGJhuSxBueapWc9HJ4cJQMqT2PPQX6rhTbKnXkebsaravnLo",
					nil, errors.New("set root error"))
			},
			wantErr: "unable to set root: unable to send instruction: set root error",
		},
	}
	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			executor, mockJSONRPCClient := newTestExecutor(t, auth, chainSelector)
			tt.setup(t, executor, mockJSONRPCClient)

			got, err := executor.SetRoot(ctx, tt.metadata, tt.proof, tt.root, tt.validUntil, tt.signatures)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.NotNil(t, got.RawData)
				require.Empty(t, cmp.Diff(tt.want, got.Hash))
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

	return NewExecutor(encoder, client, auth), mockJSONRPCClient
}

func TestIsAccountAlreadyInUseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "account already in use error",
			err: errors.New(`
				(string) (len=51) "Program 11111111111111111111111111111111 invoke [2]",
				(string) (len=110) "Allocate: account Address { address: DKvwoHhcRMZwsedUvJqmsNuat52WdcYQD7agqMjJybbF, base: None } already in use",
				(string) (len=74) "Program 11111111111111111111111111111111 failed: custom program error: 0x0",
				(string) (len=90) "Program 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk consumed 7202 of 200000 compute units",
			`),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New(`other error`),
			want: false,
		},
		{
			name: "other similar but not quite the same error",
			err:  errors.New(`Allocate: already in use`),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isAccountAlreadyInUseError(tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}
