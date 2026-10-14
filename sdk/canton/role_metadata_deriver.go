package canton

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.RoleMetadataDeriver = (*RoleMetadataDeriver)(nil)

// RoleMetadataDeriver derives role-specific Canton chain metadata. A single MCMS contract holds
// per-role state, selected by the role suffix of AdditionalFieldsMetadata.MultisigId
// ("<instanceId>@<party>-<role>"), so the source MCMAddress is kept and only the multisig id is
// rewritten. ChainId and InstanceId are preserved.
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
		return types.ChainMetadata{}, errors.New("missing canton additional fields")
	}

	var additionalFields AdditionalFieldsMetadata
	if err := json.Unmarshal(source.AdditionalFields, &additionalFields); err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to unmarshal canton additional fields: %w", err)
	}

	role, err := CantonRoleFromAction(action)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to resolve canton role for action %q: %w", action, err)
	}

	additionalFields.MultisigId, err = multisigIDForRole(additionalFields.MultisigId, role)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	derived := source
	derived.AdditionalFields, err = json.Marshal(additionalFields)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to marshal canton additional fields: %w", err)
	}

	return derived, nil
}

// multisigIDForRole replaces the role suffix of a "<instanceId>@<party>-<role>" multisig id with
// the given role. The suffix is matched against the known roles because instance ids and parties
// may themselves contain '-'.
func multisigIDForRole(multisigID string, role TimelockRole) (string, error) {
	for _, known := range []TimelockRole{TimelockRoleProposer, TimelockRoleCanceller, TimelockRoleBypasser} {
		suffix := multisigIDRoleSuffix(known)
		if base, ok := strings.CutSuffix(multisigID, suffix); ok && base != "" {
			return base + multisigIDRoleSuffix(role), nil
		}
	}

	return "", fmt.Errorf("canton multisigId %q does not end with a known role suffix", multisigID)
}

func multisigIDRoleSuffix(role TimelockRole) string {
	return "-" + strings.ToLower(role.String())
}
