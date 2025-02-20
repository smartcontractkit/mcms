package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ag_binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	cselectors "github.com/smartcontractkit/chain-selectors"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func Test_NewConfigurer(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth := solana.MustPrivateKeyFromBase58("DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc")
	instructionAuth := solana.MPK("7EcDhSYGxXyscszYEp35KHN8vvw3svAuLKTzXwCFLtV")
	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)

	tests := []struct {
		name                string
		constructorFn       func() *Configurer
		wantInstructionAuth solana.PublicKey
	}{
		{
			name: "implicit instruction authority",
			constructorFn: func() *Configurer {
				return NewConfigurer(client, auth, chainSelector)
			},
			wantInstructionAuth: auth.PublicKey(),
		},
		{
			name: "explicit instruction authority",
			constructorFn: func() *Configurer {
				return NewConfigurer(client, auth, chainSelector, WithInstructionAuth(instructionAuth))
			},
			wantInstructionAuth: instructionAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			configurer := tt.constructorFn()

			require.NotNil(t, configurer)
			require.Equal(t, tt.wantInstructionAuth, configurer.instructionsAuth)
		})
	}
}

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	defaultMcmConfig := &types.Config{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x1")}}
	clearRoot := false

	tests := []struct {
		name             string
		auth             solana.PrivateKey
		options          []configurerOption
		mcmConfig        *types.Config
		setup            func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient)
		wantHash         string
		wantInstructions []solana.Instruction
		wantErr          string
	}{
		{
			name:      "success - send instructions",
			auth:      auth,
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// TODO: extract/decode payload in transaction data and test values
				// 4 transactions: init-signers, append-signers, finalize-signers, set-config
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 11, 21,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
					"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 13, 23,
					"52f3VmvW7m9uTQu3PtyibgxnAvEuXDmm9umuHherGjS4pzRR7QXRDKnZhh6b95P7pQxzTgvE1muMNKYEY7YWsS3G", nil, nil)
			},
			wantHash: "52f3VmvW7m9uTQu3PtyibgxnAvEuXDmm9umuHherGjS4pzRR7QXRDKnZhh6b95P7pQxzTgvE1muMNKYEY7YWsS3G",
			wantInstructions: []solana.Instruction{
				bindings.NewInitSignersInstructionBuilder().Build(),
				bindings.NewAppendSignersInstructionBuilder().Build(),
				bindings.NewFinalizeSignersInstructionBuilder().Build(),
				bindings.NewSetConfigInstructionBuilder().Build(),
			},
		},
		{
			name:      "success - do not send instructions",
			auth:      nil,
			options:   []configurerOption{WithInstructionAuth(auth.PublicKey())},
			mcmConfig: defaultMcmConfig,
			setup:     func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) { t.Helper() },
			wantHash:  "",
			wantInstructions: []solana.Instruction{
				bindings.NewInitSignersInstructionBuilder().Build(),
				bindings.NewAppendSignersInstructionBuilder().Build(),
				bindings.NewFinalizeSignersInstructionBuilder().Build(),
				bindings.NewSetConfigInstructionBuilder().Build(),
			},
		},
		{
			name:      "failure: too many signers",
			auth:      auth,
			mcmConfig: &types.Config{Quorum: 1, Signers: generateSigners(t, 181)},
			setup:     func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) { t.Helper() },
			wantErr:   "too many signers (max 180)",
		},
		{
			name:      "failure: initialize signers error",
			auth:      auth,
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr",
					nil, fmt.Errorf("initialize signers error"))
			},
			wantErr: "unable to set config: unable to send instruction 0 - initSigners: unable to send instruction: initialize signers error",
		},
		{
			name:      "failure: append signers error",
			auth:      auth,
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// initialize signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil, nil)

				// append signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V",
					nil, fmt.Errorf("append signers error"))
			},
			wantErr: "unable to set config: unable to send instruction 1 - appendSigners0: unable to send instruction: append signers error",
		},
		{
			name:      "failure: finalize signers error",
			auth:      auth,
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// initialize signers + append signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V", nil, nil)

				// finalize signers
				mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
					"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya",
					nil, fmt.Errorf("finalize signers error"))
			},
			wantErr: "unable to set config: unable to send instruction 2 - finalizeSigners: unable to send instruction: finalize signers error",
		},
		{
			name:      "failure: set config error",
			auth:      auth,
			mcmConfig: &types.Config{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x1")}},
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				// initialize signers + append signers + finalize signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V", nil, nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
					"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya", nil, nil)

				// set config
				mockSolanaTransaction(t, mockJSONRPCClient, 13, 23,
					"52f3VmvW7m9uTQu3PtyibgxnAvEuXDmm9umuHherGjS4pzRR7QXRDKnZhh6b95P7pQxzTgvE1muMNKYEY7YWsS3G",
					nil, fmt.Errorf("set config error"))
			},
			wantErr: "unable to set config: unable to send instruction 3 - setConfig: unable to send instruction: set config error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configurer, mockJSONRPCClient := newTestConfigurer(t, tt.auth, chainSelector, tt.options...)
			tt.setup(t, configurer, mockJSONRPCClient)

			got, err := configurer.SetConfig(ctx, ContractAddress(testMCMProgramID, testPDASeed), tt.mcmConfig, clearRoot)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.wantHash, got.Hash))
				require.Empty(t, cmp.Diff(tt.wantInstructions, got.RawTransaction.([]solana.Instruction),
					cmpopts.IgnoreFields(ag_binary.BaseVariant{}, "Impl")))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

func newTestConfigurer(
	t *testing.T, auth solana.PrivateKey, chainSelector types.ChainSelector, options ...configurerOption,
) (*Configurer, *mocks.JSONRPCClient) {
	t.Helper()

	mockJSONRPCClient := mocks.NewJSONRPCClient(t)
	client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)

	fmt.Printf("AAAAAAAAAAAAAA %v\n", len(options))
	return NewConfigurer(client, auth, chainSelector, options...), mockJSONRPCClient
}
