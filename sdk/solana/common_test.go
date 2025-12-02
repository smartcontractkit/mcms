package solana

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	cselectors "github.com/smartcontractkit/chain-selectors"
	cpiStubBindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/external_program_cpi_stub"
	mcmBindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/mcm"
	timelockBindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

var (
	testChainSelector     = types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)
	testTimelockProgramID = solana.MustPublicKeyFromBase58("LoCoNsJFuhTkSQjfdDfn3yuwqhSYoPujmviRHVCzsqn")
	testMCMProgramID      = solana.MustPublicKeyFromBase58("6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX")
	testCPIStubProgramID  = solana.MustPublicKeyFromBase58("4HeqEoSyfYpeC2goFLj9eHgkxV33mR5G7JYAbRsN14uQ")
	testOpID              = [32]byte{1, 2, 3, 4}
	testPDASeed           = PDASeed{'t', 'e', 's', 't', '-', 'm', 'c', 'm'}
	testTimelockSeed      = PDASeed{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e'}
	testRoot              = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
)

func Test_FindSignerPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindSignerPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG")))
}

func Test_FindConfigPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindConfigPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("CiPYshUKNDV9i4p4MLaqXSRqYWtnMtW6b1aYjh4Lw9nP")))
}

func Test_getConfigSignersPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindConfigSignersPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("EZJdMB7TCRcSTP6KMp1HPnzwNtW6wvqKXHLAZh1Jn81w")))
}

func Test_FindRootMetadataPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindRootMetadataPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("H45XH8Z1zpcLUHLLQzUwEgB1s3mZQcRvCYfcHriRcMxN")))
}

func Test_FindExpiringRootAndOpCountPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindExpiringRootAndOpCountPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("7nh2qGybwNRxL3zKpiSUzk2yc9CjCb5MhrB61B98hYZu")))
}

func Test_FindRootSignaturesPDA(t *testing.T) {
	t.Parallel()
	authority, err := solana.PublicKeyFromBase58("LoCoNsJFuhTkSQjfdDfn3yuwqhSYoPujmviRHVCzsqn")
	require.NoError(t, err)

	pda, err := FindRootSignaturesPDA(testMCMProgramID, testPDASeed, testRoot, 1735689600, authority)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("98m4KuqTruBS7oXDm4Kzq43TmF1Lih3J6pHHZ6Lo9hty")))
}

func Test_FindSeenSignedHashesPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindSeenSignedHashesPDA(testMCMProgramID, testPDASeed, testRoot, 1735689600)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("FxPYSHG9tm35T43zpAuVDdNY8uMPQfaaVBftxVrLyXVq")))
}

func Test_FindTimelockConfigPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("GYWcPzXkdzY9DJLcbFs67phqyYzmJxeEKSTtqEoo8oKz")))
}

func Test_FindTimelockOperationPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockOperationPDA(testTimelockProgramID, testPDASeed, testOpID)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("9kmDgWeckKVoW44YEp4MByUJXxxwjjxK52o1HyqSPTru")))
}

func Test_FindTimelockBypasserOperationPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockBypasserOperationPDA(testTimelockProgramID, testPDASeed, testOpID)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("5mDicsfmjcDDUuaMkrBvWVf9fgDGmA9ahUdebSAM1Aid")))
}

func Test_FindTimelockSignerPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockSignerPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("2g4vS5Y9g5FKoBakfNTEQcoyuPxuqgiXhribGxE1Vrsb")))
}

func Test_sendAndConfirm(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	commitmentType := rpc.CommitmentConfirmed
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	mcmConfigPDA, err := FindConfigPDA(testMCMProgramID, testPDASeed)
	require.NoError(t, err)
	timelockConfigPDA, err := FindConfigPDA(testTimelockProgramID, testPDASeed)
	require.NoError(t, err)

	tests := []struct {
		name            string
		setup           func(*mocks.JSONRPCClient)
		builder         any
		opts            []sendTransactionOption
		wantSignature   string
		wantTransaction *rpc.GetTransactionResult
		wantErr         string
	}{
		{
			name:    "success: mcm instruction",
			builder: mcmBindings.NewAcceptOwnershipInstruction(testPDASeed, mcmConfigPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"KCXrjxMUkZ8mYmYB8uTKyCzHuAEHFwgRy7McRsrSPA9MndPjkPtsc2zA82ZKh9mBxB41REzghVMCTGLuNqWkzhp",
					ptrTo(solana.UnixTimeSeconds(1735689600)), nil)
			},
			wantSignature: "KCXrjxMUkZ8mYmYB8uTKyCzHuAEHFwgRy7McRsrSPA9MndPjkPtsc2zA82ZKh9mBxB41REzghVMCTGLuNqWkzhp",
			wantTransaction: &rpc.GetTransactionResult{
				Slot:        20,
				BlockTime:   ptrTo(solana.UnixTimeSeconds(1735689600)),
				Transaction: buildTransactionEnvelope(t, "KCXrjxMUkZ8mYmYB8uTKyCzHuAEHFwgRy7McRsrSPA9MndPjkPtsc2zA82ZKh9mBxB41REzghVMCTGLuNqWkzhp"),
				Meta:        &rpc.TransactionMeta{},
				Version:     1,
			},
		},
		{
			name:    "success: timelock instruction",
			builder: timelockBindings.NewAcceptOwnershipInstruction(testPDASeed, timelockConfigPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 30,
					"3GfwA3NeeTATEUoSjySsLGfhDX2NAhbfzrcPqEofE6fYbyRtxHXQTcprmxC8wfenck624UXXkd2NHNeBU6Qe7z3t",
					ptrTo(solana.UnixTimeSeconds(1735689600)), nil)
			},
			wantSignature: "3GfwA3NeeTATEUoSjySsLGfhDX2NAhbfzrcPqEofE6fYbyRtxHXQTcprmxC8wfenck624UXXkd2NHNeBU6Qe7z3t",
			wantTransaction: &rpc.GetTransactionResult{
				Slot:        30,
				BlockTime:   ptrTo(solana.UnixTimeSeconds(1735689600)),
				Transaction: buildTransactionEnvelope(t, "3GfwA3NeeTATEUoSjySsLGfhDX2NAhbfzrcPqEofE6fYbyRtxHXQTcprmxC8wfenck624UXXkd2NHNeBU6Qe7z3t"),
				Meta:        &rpc.TransactionMeta{},
				Version:     1,
			},
		},
		{
			name:    "failure: unsupported instruction builder error",
			builder: cpiStubBindings.NewEmptyInstruction(),
			setup:   func(mockJSONRPCClient *mocks.JSONRPCClient) {},
			wantErr: "unable to validate and build instruction: unsupported instruction builder: ",
		},
		{
			name:    "failure: ValidateAndBuild error",
			builder: &invalidTestInstruction{},
			setup:   func(mockJSONRPCClient *mocks.JSONRPCClient) {},
			wantErr: "unable to validate and build instruction: validate and build error",
		},
		{
			name:    "failure: sendAndConfirm error",
			builder: mcmBindings.NewAcceptOwnershipInstruction(testPDASeed, mcmConfigPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh",
					nil, errors.New("send and confirm error"))
			},
			opts:    []sendTransactionOption{WithRetries(1)},
			wantErr: "unable to send instruction: send and confirm error",
		},
		{
			name:    "failure: sendAndConfirm error - nil Meta",
			builder: mcmBindings.NewAcceptOwnershipInstruction(testPDASeed, mcmConfigPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh",
					nil, nil, withSolanaTransactionMeta(nil))
			},
			opts:    []sendTransactionOption{WithRetries(1)},
			wantErr: "transaction meta is nil for transaction",
		},
		{
			name:    "failure: sendAndConfirm error - Meta.Err",
			builder: mcmBindings.NewAcceptOwnershipInstruction(testPDASeed, mcmConfigPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				txMeta := &rpc.TransactionMeta{
					Err: errors.New("solana transaction meta error"),
				}

				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh",
					nil, nil, withSolanaTransactionMeta(txMeta))
			},
			opts:    []sendTransactionOption{WithRetries(1)},
			wantErr: "unable to send instruction: transaction failed: &{Err:solana transaction meta error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockJSONRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)
			tt.setup(mockJSONRPCClient)

			gotSignature, gotTransaction, err := sendAndConfirm(ctx, client, auth, tt.builder, commitmentType, tt.opts...)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.wantSignature, gotSignature)
				require.Equal(t, tt.wantTransaction, gotTransaction)
			} else {
				require.Empty(t, gotSignature)
				require.Nil(t, gotTransaction)
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

type invalidTestInstruction struct{}

func (*invalidTestInstruction) ValidateAndBuild() (*mcmBindings.Instruction, error) {
	return nil, errors.New("validate and build error ")
}

func buildTransactionEnvelope(t *testing.T, signature string) *rpc.TransactionResultEnvelope {
	t.Helper()

	var transactionEnvelope rpc.TransactionResultEnvelope
	err := transactionEnvelope.UnmarshalJSON([]byte(`{
		"signatures": ["` + signature + `"],
		"message": {}
	}`))
	require.NoError(t, err)

	return &transactionEnvelope
}
