package ton

import (
	"context"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.RoleMetadataDeriver = (*RoleMetadataDeriver)(nil)

// RoleMetadataDeriver derives role-specific chain metadata. Each timelock role is held by a
// separate MCM contract, so the role's MCM address is resolved via the RoleAddressResolver and
// AdditionalFields are kept unchanged.
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
