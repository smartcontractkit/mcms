package chainwrappers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/chainwrappers/mocks"
	"github.com/smartcontractkit/mcms/internal/testutils/chaintest"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func TestBuildRoleMetadataDeriver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   types.ChainSelector
		expectType any
		expectErr  string
	}{
		{name: "evm", selector: chaintest.Chain2Selector, expectType: (*evm.RoleMetadataDeriver)(nil)},
		{name: "solana", selector: chaintest.Chain4Selector, expectType: (*solanasdk.RoleMetadataDeriver)(nil)},
		{name: "aptos", selector: chaintest.Chain5Selector, expectType: (*aptos.RoleMetadataDeriver)(nil)},
		{name: "sui", selector: chaintest.Chain6Selector, expectType: (*sui.RoleMetadataDeriver)(nil)},
		{name: "unsupported family", selector: chaintest.Chain8Selector, expectErr: "unsupported chain family"},
		{name: "invalid selector", selector: chaintest.ChainInvalidSelector, expectErr: "error getting chain family"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deriver, err := BuildRoleMetadataDeriver(nil, tc.selector)
			if tc.expectErr != "" {
				require.ErrorContains(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tc.expectType, deriver)
		})
	}
}

func TestBuildRoleMetadataDeriver_SolanaExecutePayerFromChainAccessor(t *testing.T) {
	t.Parallel()

	signer, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	access := mocks.NewChainAccessor(t)
	access.EXPECT().SolanaSigner(mock.Anything).Return(&signer, true)

	deriver, err := BuildRoleMetadataDeriver(access, chaintest.Chain4Selector)
	require.NoError(t, err)

	resolver := sdk.RoleAddressResolverFunc(func(context.Context, sdk.RoleAddressRequest) (string, error) {
		return "bypasser.seed", nil
	})
	sourceFields, err := json.Marshal(solanasdk.AdditionalFieldsMetadata{
		ProposerRoleAccessController:  solana.NewWallet().PublicKey(),
		CancellerRoleAccessController: solana.NewWallet().PublicKey(),
		BypasserRoleAccessController:  solana.NewWallet().PublicKey(),
	})
	require.NoError(t, err)
	got, err := deriver.DeriveRoleMetadata(t.Context(), chaintest.Chain4Selector, types.ChainMetadata{AdditionalFields: sourceFields}, types.TimelockActionBypass, resolver)
	require.NoError(t, err)

	var fields solanasdk.AdditionalFieldsMetadata
	require.NoError(t, json.Unmarshal(got.AdditionalFields, &fields))
	require.NotNil(t, fields.ExecutePayer)
	require.Equal(t, signer.PublicKey(), *fields.ExecutePayer)
}

func TestBuildRoleMetadataDeriver_SolanaWithoutSigner(t *testing.T) {
	t.Parallel()

	access := mocks.NewChainAccessor(t)
	access.EXPECT().SolanaSigner(mock.Anything).Return(nil, false)

	deriver, err := BuildRoleMetadataDeriver(access, chaintest.Chain4Selector)
	require.NoError(t, err)

	resolver := sdk.RoleAddressResolverFunc(func(context.Context, sdk.RoleAddressRequest) (string, error) {
		return "bypasser.seed", nil
	})
	_, err = deriver.DeriveRoleMetadata(t.Context(), chaintest.Chain4Selector, types.ChainMetadata{}, types.TimelockActionBypass, resolver)
	require.ErrorContains(t, err, "solana execute payer is required")
}
