package canton

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

func TestRoleMetadataDeriver_DeriveRoleMetadata(t *testing.T) {
	t.Parallel()

	const (
		instanceAddress = "0x" + "ab00000000000000000000000000000000000000000000000000000000000000"
		instanceID      = "mcms-ccip"
		party           = "mcms-owner::1220abcdef"
	)
	metadataFor := func(t *testing.T, role string) types.ChainMetadata {
		t.Helper()
		md, err := NewChainMetadata(4, 7, instanceID+"@"+party+"-"+role, instanceAddress, instanceID)
		require.NoError(t, err)

		return md
	}
	failingResolver := sdk.RoleAddressResolverFunc(func(context.Context, sdk.RoleAddressRequest) (string, error) {
		t.Fatal("canton must not consult the role address resolver")
		return "", nil
	})

	tests := []struct {
		name       string
		sourceRole string
		action     types.TimelockAction
		fields     json.RawMessage
		wantRole   string
		wantErr    string
	}{
		{name: "proposer to canceller", sourceRole: "proposer", action: types.TimelockActionCancel, wantRole: "canceller"},
		{name: "proposer to bypasser", sourceRole: "proposer", action: types.TimelockActionBypass, wantRole: "bypasser"},
		{name: "proposer to proposer", sourceRole: "proposer", action: types.TimelockActionSchedule, wantRole: "proposer"},
		{name: "canceller to bypasser", sourceRole: "canceller", action: types.TimelockActionBypass, wantRole: "bypasser"},
		{name: "unknown role suffix", sourceRole: "admin", action: types.TimelockActionCancel, wantErr: "does not end with a known role suffix"},
		{name: "missing additional fields", sourceRole: "proposer", action: types.TimelockActionCancel, fields: json.RawMessage{}, wantErr: "missing canton additional fields"},
		{name: "invalid additional fields", sourceRole: "proposer", action: types.TimelockActionCancel, fields: json.RawMessage(`{`), wantErr: "unable to unmarshal canton additional fields"},
		{name: "unknown action", sourceRole: "proposer", action: types.TimelockAction("unknown"), wantErr: "failed to resolve canton role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := metadataFor(t, tt.sourceRole)
			if tt.fields != nil {
				source.AdditionalFields = tt.fields
			}

			got, err := NewRoleMetadataDeriver().DeriveRoleMetadata(t.Context(), 1, source, tt.action, failingResolver)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, metadataFor(t, tt.wantRole), got)
		})
	}
}

func TestMultisigIDForRole(t *testing.T) {
	t.Parallel()

	got, err := multisigIDForRole("mcms-proposer@party-proposer", TimelockRoleCanceller)
	require.NoError(t, err)
	require.Equal(t, "mcms-proposer@party-canceller", got)

	_, err = multisigIDForRole("-proposer", TimelockRoleCanceller)
	require.ErrorContains(t, err, "known role suffix")

	_, err = multisigIDForRole(strings.ToUpper("mcms@party-proposer"), TimelockRoleCanceller)
	require.ErrorContains(t, err, "known role suffix")
}
