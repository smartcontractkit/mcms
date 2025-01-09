package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

var (
	testProgramID = solana.MustPublicKeyFromBase58("6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX")
	testPDASeed   = PDASeed{'t', 'e', 's', 't', '-', 'm', 'c', 'm'}
	testRoot      = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
)

func Test_FindSignerPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindSignerPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("62gDM6BRLf2w1yXfmpePUTsuvbeBbu4QqdjV32wcc4UG")))
}

func Test_FindConfigPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindConfigPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("CiPYshUKNDV9i4p4MLaqXSRqYWtnMtW6b1aYjh4Lw9nP")))
}

func Test_getConfigSignersPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindConfigSignersPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("EZJdMB7TCRcSTP6KMp1HPnzwNtW6wvqKXHLAZh1Jn81w")))
}

func Test_FindRootMetadataPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindRootMetadataPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("H45XH8Z1zpcLUHLLQzUwEgB1s3mZQcRvCYfcHriRcMxN")))
}

func Test_FindExpiringRootAndOpCountPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindExpiringRootAndOpCountPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("7nh2qGybwNRxL3zKpiSUzk2yc9CjCb5MhrB61B98hYZu")))
}

func Test_FindRootSignaturesPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindRootSignaturesPDA(testProgramID, testPDASeed, testRoot, 1735689600)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("528jBx5Mn1EPt4vG47CRkr1zhj8QVfSMvfvBZksZdrHr")))
}

func Test_FindSeenSignedHashesPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindSeenSignedHashesPDA(testProgramID, testPDASeed, testRoot, 1735689600)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("FxPYSHG9tm35T43zpAuVDdNY8uMPQfaaVBftxVrLyXVq")))
}

func Test_FindTimelockConfigPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockConfigPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("27X4nnwKaRk93RamRXQSfNyuB1pBSSK1hf2ULUeL1VCp")))
}

func Test_FindTimelockOperationPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockOperationPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("8TL4xwjpntLQXeFbADMPnooDofGUwocc4ikHAJb41Fcm")))
}

func Test_FindTimelockSignerPDA(t *testing.T) {
	t.Parallel()
	pda, err := FindTimelockSignerPDA(testProgramID, testPDASeed)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(pda, solana.MustPublicKeyFromBase58("HAQoFdsmxUFgAfBb6u9AXvg9q1nJthWb7xMYWpvzFJfg")))
}

func Test_sendAndConfirm(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	commitmentType := rpc.CommitmentConfirmed
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	configPDA, err := FindConfigPDA(testProgramID, testPDASeed)
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		builder instructionBuilder
		want    string
		wantErr string
	}{
		{
			name:    "success",
			builder: bindings.NewAcceptOwnershipInstruction(testPDASeed, configPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"KCXrjxMUkZ8mYmYB8uTKyCzHuAEHFwgRy7McRsrSPA9MndPjkPtsc2zA82ZKh9mBxB41REzghVMCTGLuNqWkzhp", nil)
			},
			want: "KCXrjxMUkZ8mYmYB8uTKyCzHuAEHFwgRy7McRsrSPA9MndPjkPtsc2zA82ZKh9mBxB41REzghVMCTGLuNqWkzhp",
		},
		{
			name:    "failure: ValidateAndBuild error ",
			builder: &invalidTestInstruction{},
			setup:   func(mockJSONRPCClient *mocks.JSONRPCClient) {},
			wantErr: "unable to validate and build instruction: validate and build error",
		},
		{
			name:    "failure: sendAndConfirm error ",
			builder: bindings.NewAcceptOwnershipInstruction(testPDASeed, configPDA, auth.PublicKey()),
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh",
					fmt.Errorf("send and confirm error"))
			},
			wantErr: "unable to send instruction: send and confirm error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockJSONRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)
			tt.setup(mockJSONRPCClient)

			got, err := sendAndConfirm(ctx, client, auth, tt.builder, commitmentType)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

type invalidTestInstruction struct{}

func (*invalidTestInstruction) ValidateAndBuild() (*bindings.Instruction, error) {
	return nil, fmt.Errorf("validate and build error ")
}
