package solana

import (
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func Test_NewConfigurer(t *testing.T) {
	t.Parallel()

	client := &rpc.Client{}
	auth := solana.MustPrivateKeyFromBase58("DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc")
	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)

	configurer := NewConfigurer(client, auth, chainSelector)

	require.NotNil(t, configurer)
}

func TestConfigurer_SetConfig(t *testing.T) {
	t.Parallel()

	chainSelector := types.ChainSelector(cselectors.SOLANA_DEVNET.Selector)
	auth, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	mcmAddress := solana.MustPublicKeyFromBase58("6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX")
	configPDA := configPDA(t, mcmAddress.String())
	defaultMcmConfig := &types.Config{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x1")}}
	clearRoot := false

	tests := []struct {
		name      string
		mcmConfig *types.Config
		setup     func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient)
		want      string
		wantErr   string
	}{
		{
			name:      "success",
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				accountInfo := &mcm.MultisigConfig{ChainId: uint64(chainSelector), Owner: solana.SystemProgramID}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, accountInfo, nil)

				// TODO: extract/decode payload in transaction data and test values
				// 4 transactions: init-signers, append-signers, finalize-signers, set-config
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 11, 21,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
					"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 13, 23,
					"52f3VmvW7m9uTQu3PtyibgxnAvEuXDmm9umuHherGjS4pzRR7QXRDKnZhh6b95P7pQxzTgvE1muMNKYEY7YWsS3G", nil)
			},
			want: "52f3VmvW7m9uTQu3PtyibgxnAvEuXDmm9umuHherGjS4pzRR7QXRDKnZhh6b95P7pQxzTgvE1muMNKYEY7YWsS3G",
		},
		{
			name:      "failure: too many signers",
			mcmConfig: &types.Config{Quorum: 1, Signers: generateSigners(t, math.MaxUint8+1)},
			setup:     func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) { t.Helper() },
			wantErr:   "too many signers (max 255)",
		},
		{
			name:      "failure: initialize signers error",
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				accountInfo := &mcm.MultisigConfig{ChainId: uint64(chainSelector), Owner: solana.SystemProgramID}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, accountInfo, nil)

				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr",
					fmt.Errorf("initialize signers error"))
			},
			wantErr: "unable to initialize signers: unable to send instruction: initialize signers error",
		},
		{
			name:      "failure: append signers error",
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				accountInfo := &mcm.MultisigConfig{ChainId: uint64(chainSelector), Owner: solana.SystemProgramID}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, accountInfo, nil)

				// initialize signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil)

				// append signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V",
					fmt.Errorf("append signers error"))
			},
			wantErr: "unable to append signers (0): unable to send instruction: append signers error",
		},
		{
			name:      "failure: finalize signers error",
			mcmConfig: defaultMcmConfig,
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				accountInfo := &mcm.MultisigConfig{ChainId: uint64(chainSelector), Owner: solana.SystemProgramID}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, accountInfo, nil)

				// initialize signers + append signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V", nil)

				// finalize signers
				mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
					"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya",
					fmt.Errorf("finalize signers error"))
			},
			wantErr: "unable to finalize signers: unable to send instruction: finalize signers error",
		},
		{
			name:      "failure: set config error",
			mcmConfig: &types.Config{Quorum: 1, Signers: []common.Address{common.HexToAddress("0x1")}},
			setup: func(t *testing.T, configurer *Configurer, mockJSONRPCClient *mocks.JSONRPCClient) {
				t.Helper()

				accountInfo := &mcm.MultisigConfig{ChainId: uint64(chainSelector), Owner: solana.SystemProgramID}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, accountInfo, nil)

				// initialize signers + append signers + finalize signers
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"4PQcRHQJT4cRQZooAhZMAP9ZXJsAka9DeKvXeYvXAvPpHb4Qkc5rmTSHDA2SZSh9aKPBguBx4kmcyHHbkytoAiRr", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 10, 20,
					"7D9XEYRnCn1D5JFrrYMPUaHfog7Vnj5rbPdj7kbULa4hKq7GsnA7Q8KNQfLEgfCawBsW4dcH2MQAp4km1dnjr6V", nil)
				mockSolanaTransaction(t, mockJSONRPCClient, 12, 22,
					"2iEeniu3QUgXNsjau8r7fZ7XLb2g1F3q9VJJKvRyyFz4hHgVvhGkLgSUdmRumfXKWv8spJ9ihudGFyPZsPGdp4Ya", nil)

				// set config
				mockSolanaTransaction(t, mockJSONRPCClient, 13, 23,
					"52f3VmvW7m9uTQu3PtyibgxnAvEuXDmm9umuHherGjS4pzRR7QXRDKnZhh6b95P7pQxzTgvE1muMNKYEY7YWsS3G",
					fmt.Errorf("set config error"))
			},
			wantErr: "unable to set config: unable to send instruction: set config error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configurer, mockJSONRPCClient := newTestConfigurer(t, auth, chainSelector)
			tt.setup(t, configurer, mockJSONRPCClient)

			got, err := configurer.SetConfig(mcmAddress.String(), tt.mcmConfig, clearRoot)

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

func newTestConfigurer(t *testing.T, auth solana.PrivateKey, chainSelector types.ChainSelector) (*Configurer, *mocks.JSONRPCClient) {
	t.Helper()

	mockJSONRPCClient := mocks.NewJSONRPCClient(t)
	client := rpc.NewWithCustomRPCClient(mockJSONRPCClient)

	return NewConfigurer(client, auth, chainSelector), mockJSONRPCClient
}
