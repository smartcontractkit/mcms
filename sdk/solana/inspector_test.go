package solana

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/go-cmp/cmp"
	cselectors "github.com/smartcontractkit/chain-selectors"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk/solana/mocks"
	"github.com/smartcontractkit/mcms/types"
)

func Test_NewInspector(t *testing.T) {
	t.Parallel()

	inspector := NewInspector(&rpc.Client{})
	require.NotNil(t, inspector)
}

func TestInspector_GetConfig(t *testing.T) {
	t.Parallel()

	chainSelector := cselectors.SOLANA_DEVNET.Selector
	mcmAddress := "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX"
	mcmName := [32]byte{1, 2, 3, 4}
	configPDA := configPDA(t, mcmAddress, mcmName)

	tests := []struct {
		name    string
		setup   func(*mocks.JSONRPCClient)
		want    *types.Config
		wantErr string
	}{
		{
			name: "success",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mcmConfig := &bindings.MultisigConfig{
					ChainId:      chainSelector,
					MultisigName: mcmName,
					Owner:        solana.SystemProgramID,
					Signers: []bindings.McmSigner{
						{EvmAddress: common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"), Index: 0, Group: 0},
						{EvmAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"), Index: 1, Group: 0},
						{EvmAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"), Index: 2, Group: 0},
						{EvmAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"), Index: 0, Group: 1},
						{EvmAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"), Index: 1, Group: 1},
						{EvmAddress: common.HexToAddress("0x3333333333333333333333333333333333333333"), Index: 2, Group: 1},
					},
					GroupQuorums: [32]uint8{3, 2}, // Valid configuration
					GroupParents: [32]uint8{0, 0},
				}

				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, mcmConfig, nil)
			},
			want: &types.Config{
				Quorum: 3,
				Signers: []common.Address{
					common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdef"),
					common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
					common.HexToAddress("0x1234567890abcdef1234567890abcdef56785678"),
				},
				GroupSigners: []types.Config{
					{
						Quorum: 2,
						Signers: []common.Address{
							common.HexToAddress("0x1111111111111111111111111111111111111111"),
							common.HexToAddress("0x2222222222222222222222222222222222222222"),
							common.HexToAddress("0x3333333333333333333333333333333333333333"),
						},
						GroupSigners: []types.Config{},
					},
				},
			},
		},
		{
			name: "error: rpc error",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				err := fmt.Errorf("json rpc call failed")
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, &bindings.MultisigConfig{}, err)
			},
			want:    nil,
			wantErr: "json rpc call failed",
		},
		{
			name: "error: empty signers list",
			setup: func(mockJSONRPCClient *mocks.JSONRPCClient) {
				mcmConfig := &bindings.MultisigConfig{
					Signers:      []bindings.McmSigner{},
					GroupQuorums: [32]uint8{3, 2},
					GroupParents: [32]uint8{0, 0},
				}
				mockGetAccountInfo(t, mockJSONRPCClient, configPDA, mcmConfig, nil)
			},
			want:    nil,
			wantErr: "invalid MCMS config: Quorum must be less than or equal to the number of signers and groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inspector, jsonRPCClient := newTestInspector(t)
			tt.setup(jsonRPCClient)

			got, err := inspector.GetConfig(context.Background(), mcmAddress)

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

func newTestInspector(t *testing.T) (*Inspector, *mocks.JSONRPCClient) {
	t.Helper()
	jsonRPCClient := mocks.NewJSONRPCClient(t)
	inspector := NewInspector(rpc.NewWithCustomRPCClient(jsonRPCClient))

	return inspector, jsonRPCClient
}
