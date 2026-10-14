package evm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

func TestRoleMetadataDeriver_DeriveRoleMetadata(t *testing.T) {
	t.Parallel()

	source := types.ChainMetadata{
		StartingOpCount:  1,
		MCMAddress:       "0xproposer",
		AdditionalFields: json.RawMessage(`{"gasLimit":100}`),
	}
	var requested []sdk.RoleAddressRequest
	resolver := sdk.RoleAddressResolverFunc(func(_ context.Context, req sdk.RoleAddressRequest) (string, error) {
		requested = append(requested, req)
		return "0x" + req.Role.String(), nil
	})

	got, err := NewRoleMetadataDeriver().DeriveRoleMetadata(t.Context(), 1, source, types.TimelockActionCancel, resolver)
	require.NoError(t, err)
	require.Equal(t, "0xCanceller", got.MCMAddress)
	require.Equal(t, source.StartingOpCount, got.StartingOpCount)
	require.JSONEq(t, string(source.AdditionalFields), string(got.AdditionalFields))

	got, err = NewRoleMetadataDeriver().DeriveRoleMetadata(t.Context(), 1, source, types.TimelockActionBypass, resolver)
	require.NoError(t, err)
	require.Equal(t, "0xBypasser", got.MCMAddress)
	require.Equal(t, []sdk.RoleAddressRequest{
		{Selector: 1, Role: sdk.TimelockRoleCanceller, SourceMCMAddress: "0xproposer"},
		{Selector: 1, Role: sdk.TimelockRoleBypasser, SourceMCMAddress: "0xproposer"},
	}, requested)

	_, err = NewRoleMetadataDeriver().DeriveRoleMetadata(t.Context(), 1, source, types.TimelockActionCancel, nil)
	require.EqualError(t, err, "role address resolver is required to resolve the Canceller MCM address")
}
