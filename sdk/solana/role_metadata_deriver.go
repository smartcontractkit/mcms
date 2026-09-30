package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.RoleMetadataDeriver = (*RoleMetadataDeriver)(nil)

// RoleMetadataDeriver derives role-specific Solana chain metadata. Each timelock role is held by a
// separate MCM instance, so the role's MCM address is resolved via the RoleAddressResolver. For
// bypass, the ExecutePayer is also recorded in AdditionalFields (see AdditionalFieldsMetadata).
type RoleMetadataDeriver struct {
	executePayer *solana.PublicKey
}

// NewRoleMetadataDeriver returns a new RoleMetadataDeriver. executePayer is the public key that
// pays for (and signs) the MCM execute transactions; it is required to derive bypass metadata.
func NewRoleMetadataDeriver(executePayer *solana.PublicKey) *RoleMetadataDeriver {
	return &RoleMetadataDeriver{executePayer: executePayer}
}

// DeriveRoleMetadata implements sdk.RoleMetadataDeriver.
func (d *RoleMetadataDeriver) DeriveRoleMetadata(
	ctx context.Context,
	selector types.ChainSelector,
	source types.ChainMetadata,
	action types.TimelockAction,
	resolver sdk.RoleAddressResolver,
) (types.ChainMetadata, error) {
	derived, err := sdk.ResolveRoleMetadata(ctx, selector, source, action, resolver)
	if err != nil {
		return types.ChainMetadata{}, err
	}
	if action != types.TimelockActionBypass {
		return derived, nil
	}

	if d.executePayer == nil {
		return types.ChainMetadata{}, errors.New("solana execute payer is required to derive bypass metadata")
	}

	var additionalFields AdditionalFieldsMetadata
	if len(derived.AdditionalFields) > 0 {
		if err = json.Unmarshal(derived.AdditionalFields, &additionalFields); err != nil {
			return types.ChainMetadata{}, fmt.Errorf("unable to unmarshal solana additional fields: %w", err)
		}
	}
	derived.AdditionalFields, err = json.Marshal(additionalFields.WithExecutePayer(*d.executePayer))
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("unable to marshal solana additional fields: %w", err)
	}

	return derived, nil
}
