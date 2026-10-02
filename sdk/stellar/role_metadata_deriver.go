package stellar

import (
	"context"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.RoleMetadataDeriver = (*RoleMetadataDeriver)(nil)

// RoleMetadataDeriver derives role-specific Stellar chain metadata. Stellar registers role-specific
// aliases for its MCMS address, so the role address is resolved via RoleAddressResolver while
// AdditionalFields remain unchanged.
type RoleMetadataDeriver struct{}

// NewRoleMetadataDeriver returns a new RoleMetadataDeriver.
func NewRoleMetadataDeriver() *RoleMetadataDeriver {
	return &RoleMetadataDeriver{}
}

// DeriveRoleMetadata implements sdk.RoleMetadataDeriver.
func (d *RoleMetadataDeriver) DeriveRoleMetadata(
	ctx context.Context,
	selector types.ChainSelector,
	source types.ChainMetadata,
	action types.TimelockAction,
	resolver sdk.RoleAddressResolver,
) (types.ChainMetadata, error) {
	return sdk.ResolveRoleMetadata(ctx, selector, source, action, resolver)
}
