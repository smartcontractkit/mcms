package sui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.RoleMetadataDeriver = (*RoleMetadataDeriver)(nil)

// RoleMetadataDeriver derives role-specific Sui chain metadata. A single MCMS object fulfills every
// timelock role, selected by AdditionalFieldsMetadata.Role, so the source MCMAddress is kept and
// only the role is rewritten. All object IDs are preserved.
type RoleMetadataDeriver struct{}

// NewRoleMetadataDeriver returns a new RoleMetadataDeriver.
func NewRoleMetadataDeriver() *RoleMetadataDeriver {
	return &RoleMetadataDeriver{}
}

// DeriveRoleMetadata implements sdk.RoleMetadataDeriver. resolver is not consulted.
func (d *RoleMetadataDeriver) DeriveRoleMetadata(
	_ context.Context,
	_ types.ChainSelector,
	source types.ChainMetadata,
	action types.TimelockAction,
	_ sdk.RoleAddressResolver,
) (types.ChainMetadata, error) {
	if len(source.AdditionalFields) == 0 {
		return types.ChainMetadata{}, errors.New("missing sui additional fields")
	}

	var additionalFields AdditionalFieldsMetadata
	if err := json.Unmarshal(source.AdditionalFields, &additionalFields); err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to unmarshal sui additional fields: %w", err)
	}

	role, err := SuiRoleFromAction(action)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to resolve sui role for action %q: %w", action, err)
	}
	additionalFields.Role = role

	derived := source
	derived.AdditionalFields, err = json.Marshal(additionalFields)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to marshal sui additional fields: %w", err)
	}

	return derived, nil
}
