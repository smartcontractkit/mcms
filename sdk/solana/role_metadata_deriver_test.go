package solana

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

func TestRoleMetadataDeriver_DeriveRoleMetadata(t *testing.T) {
	t.Parallel()

	payer := solana.NewWallet().PublicKey()
	sourceFields := AdditionalFieldsMetadata{
		ProposerRoleAccessController:  solana.NewWallet().PublicKey(),
		CancellerRoleAccessController: solana.NewWallet().PublicKey(),
		BypasserRoleAccessController:  solana.NewWallet().PublicKey(),
	}
	rawSourceFields, err := json.Marshal(sourceFields)
	require.NoError(t, err)

	roleAddresses := map[sdk.TimelockRole]string{
		sdk.TimelockRoleCanceller: "canceller.seed",
		sdk.TimelockRoleBypasser:  "bypasser.seed",
	}
	resolver := sdk.RoleAddressResolverFunc(func(_ context.Context, req sdk.RoleAddressRequest) (string, error) {
		return roleAddresses[req.Role], nil
	})

	tests := []struct {
		name       string
		payer      *solana.PublicKey
		action     types.TimelockAction
		fields     json.RawMessage
		wantAddr   string
		wantFields AdditionalFieldsMetadata
		wantErr    string
	}{
		{
			name:       "cancel keeps additional fields",
			payer:      &payer,
			action:     types.TimelockActionCancel,
			fields:     rawSourceFields,
			wantAddr:   "canceller.seed",
			wantFields: sourceFields,
		},
		{
			name:       "bypass sets execute payer",
			payer:      &payer,
			action:     types.TimelockActionBypass,
			fields:     rawSourceFields,
			wantAddr:   "bypasser.seed",
			wantFields: sourceFields.WithExecutePayer(payer),
		},
		{
			name:    "bypass rejects empty additional fields",
			payer:   &payer,
			action:  types.TimelockActionBypass,
			wantErr: "unable to unmarshal solana additional fields",
		},
		{
			name:       "cancel does not require payer",
			action:     types.TimelockActionCancel,
			fields:     rawSourceFields,
			wantAddr:   "canceller.seed",
			wantFields: sourceFields,
		},
		{
			name:    "bypass requires payer",
			action:  types.TimelockActionBypass,
			fields:  rawSourceFields,
			wantErr: "solana execute payer is required",
		},
		{
			name:    "bypass rejects zero payer",
			payer:   &solana.PublicKey{},
			action:  types.TimelockActionBypass,
			fields:  rawSourceFields,
			wantErr: "solana execute payer is required",
		},
		{
			name:    "bypass with invalid additional fields",
			payer:   &payer,
			action:  types.TimelockActionBypass,
			fields:  json.RawMessage(`{`),
			wantErr: "unable to unmarshal solana additional fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := types.ChainMetadata{StartingOpCount: 2, MCMAddress: "proposer.seed", AdditionalFields: tt.fields}
			got, err := NewRoleMetadataDeriver(tt.payer).DeriveRoleMetadata(t.Context(), 1, source, tt.action, resolver)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantAddr, got.MCMAddress)
			require.Equal(t, source.StartingOpCount, got.StartingOpCount)

			var derived AdditionalFieldsMetadata
			if len(got.AdditionalFields) > 0 {
				require.NoError(t, json.Unmarshal(got.AdditionalFields, &derived))
			}
			require.Equal(t, tt.wantFields, derived)
		})
	}
}
