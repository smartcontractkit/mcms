package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
)

func Test_initializeMcmProgram(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	chainSelector := cselectors.SOLANA_DEVNET.Selector
	mcmAddress := "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX"
	mcmAddressPubKey := solana.MustPublicKeyFromBase58(mcmAddress)
	configPDA := configPDA(t, mcmAddress)
	rootMetadataPDA := rootMetadataPDA(t, mcmAddress)
	expiringRootAndOpCountPDA := expiringRootAndOpCountPDA(t, mcmAddress)
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		wantErr string
	}{
		{
			name: "success: already initialized",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mcmConfig := &bindings.MultisigConfig{
					ChainId:      chainSelector,
					MultisigName: mcmName,
					Owner:        solana.SystemProgramID,
				}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, mcmConfig, nil)
			},
		},
		{
			name: "success: not initialized",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := fmt.Errorf("already initialized")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)

				data := struct {
					DataType uint32
					Address  solana.PublicKey
				}{1, mcmAddressPubKey}
				mockGetAccountInfo(t, mockJSONRPCClient, mcmAddressPubKey, data, nil)

				// mock NewInitializeInstruction call
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh", nil)

				mcmConfig := &bindings.MultisigConfig{
					ChainId:      chainSelector,
					MultisigName: mcmName,
					Owner:        auth.PublicKey(),
				}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, mcmConfig, nil)
			},
		},
		{
			name: "failure: mcm program GetAccountInfo error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := fmt.Errorf("already initialized")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)

				err = fmt.Errorf("get account info error")
				mockGetAccountInfo(t, mockJSONRPCClient, mcmAddressPubKey, nil, err)
			},
			wantErr: "get account info error",
		},
		{
			name: "failure: unmarshal error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := fmt.Errorf("already initialized")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)

				data := struct{ Invalid string }{}
				mockGetAccountInfo(t, mockJSONRPCClient, mcmAddressPubKey, data, nil)
			},
			wantErr: "unable to unmarshal borsh: error while decoding \"Address\" field",
		},
		{
			name: "failure: send transaction error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := fmt.Errorf("already initialized")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)

				data := struct {
					DataType uint32
					Address  solana.PublicKey
				}{1, mcmAddressPubKey}
				mockGetAccountInfo(t, mockJSONRPCClient, mcmAddressPubKey, data, nil)

				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh",
					fmt.Errorf("get latest blockhash error"))
			},
			wantErr: "get latest blockhash error",
		},
		{
			name: "failure: configpda confirmation get account info error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := fmt.Errorf("already initialized")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)

				data := struct {
					DataType uint32
					Address  solana.PublicKey
				}{1, mcmAddressPubKey}
				mockGetAccountInfo(t, mockJSONRPCClient, mcmAddressPubKey, data, nil)

				// mock NewInitializeInstruction call
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"NyH6sKKEbAMjxzG9qLTcwd1yEmv46Z94XmH5Pp9AXJps8EofvpPdUn5bp7rzKnztWmxskBiVRnp4DwaHujhHvFh", nil)

				err = fmt.Errorf("confirmation get account info error")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)
			},
			wantErr: "confirmation get account info error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockJSONRPCClient := mocks.NewJSONRPCClient(t)
			client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)
			tt.setup(mockJSONRPCClient)

			err := initializeMcmProgram(ctx, client, auth, chainSelector, mcmAddressPubKey, mcmName, configPDA,
				rootMetadataPDA, expiringRootAndOpCountPDA)

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_sendAndConfirm(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	commitmentType := rpc.CommitmentConfirmed
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	configPDA := configPDA(t, "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX")

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		builder instructionBuilder
		want    string
		wantErr string
	}{
		{
			name:    "success",
			builder: bindings.NewAcceptOwnershipInstruction(mcmName, configPDA, auth.PublicKey()),
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
			builder: bindings.NewAcceptOwnershipInstruction(mcmName, configPDA, auth.PublicKey()),
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
