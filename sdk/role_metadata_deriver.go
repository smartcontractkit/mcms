package sdk

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/mcms/types"
)

// RoleAddressRequest describes the MCM address a RoleMetadataDeriver needs from its caller: the
// MCM that holds Role on Selector, paired with the schedule proposal's MCM (SourceMCMAddress).
// Fields may be added over time; resolvers should ignore fields they do not use.
type RoleAddressRequest struct {
	Selector types.ChainSelector
	Role     TimelockRole
	// SourceMCMAddress is the MCM address of the source (schedule) proposal on Selector. Resolvers
	// can use it to pick the role MCM deployed alongside it (e.g. with the same datastore qualifier).
	SourceMCMAddress string
}

// RoleAddressResolver looks up the MCM address that holds a timelock role on a chain.
// It is implemented by callers (e.g. backed by a datastore, an address file, explicit flags or an
// on-chain lookup) and consulted by a RoleMetadataDeriver when the role is held by a different MCM
// instance than the source proposal's.
//
// Note on the current design
// The resolver is injected, rather than having callers look up every role address up front,
// because datastores do not store role MCMs uniformly across chain families: families with one MCM
// per role (EVM, Solana, TON) register CancellerManyChainMultiSig/BypasserManyChainMultiSig refs,
// while single-MCMS families (Sui, Aptos, Canton) register them inconsistently or not at all (e.g.
// with a different qualifier, see CCIP-13301). Only the chain family knows whether a lookup is
// needed, so the deriver decides whether to call the resolver.
//
// TODO: if every chain family registered its role MCMs in the same shape (single-MCMS families
// writing Proposer/Canceller/Bypasser alias refs to their one MCMS with the deployment's qualifier,
// as Stellar already does), every deriver would need the same lookup. RoleAddressResolver could
// then be dropped: callers would resolve the role MCM address themselves and pass it directly to
// DeriveRoleMetadata, which would become a pure function. This needs changes to each family's
// deployment changesets plus a backfill of existing datastores.
type RoleAddressResolver interface {
	ResolveRoleAddress(ctx context.Context, req RoleAddressRequest) (string, error)
}

// RoleAddressResolverFunc adapts an ordinary function to a RoleAddressResolver.
type RoleAddressResolverFunc func(ctx context.Context, req RoleAddressRequest) (string, error)

// ResolveRoleAddress calls f(ctx, req).
func (f RoleAddressResolverFunc) ResolveRoleAddress(ctx context.Context, req RoleAddressRequest) (string, error) {
	return f(ctx, req)
}

// RoleMetadataDeriver derives the chain metadata a proposal must carry for the given action,
// using the chain metadata of a schedule proposal (e.g. when deriving a cancellation or bypass proposal).
// Implementations own every chain-specific adjustment: they may consult resolver when the role is held
// by a different MCM instance, and rewrite AdditionalFields when the role is encoded there. The
// source metadata is not mutated and StartingOpCount is returned unchanged; callers set it
// (see chainwrappers pkg for caller implementations).
type RoleMetadataDeriver interface {
	DeriveRoleMetadata(
		ctx context.Context,
		selector types.ChainSelector,
		source types.ChainMetadata,
		action types.TimelockAction,
		resolver RoleAddressResolver,
	) (types.ChainMetadata, error)
}

// TimelockRoleFromAction returns the timelock role that performs the given action.
func TimelockRoleFromAction(action types.TimelockAction) (TimelockRole, error) {
	switch action {
	case types.TimelockActionSchedule:
		return TimelockRoleProposer, nil
	case types.TimelockActionCancel:
		return TimelockRoleCanceller, nil
	case types.TimelockActionBypass:
		return TimelockRoleBypasser, nil
	default:
		return 0, fmt.Errorf("unknown timelock action %q", action)
	}
}

// ResolveRoleMetadata is a helper for chain families whose timelock roles are held by separate MCM
// instances: it resolves the MCM address holding the role for action via resolver and returns a
// copy of source pointing at it.
func ResolveRoleMetadata(
	ctx context.Context,
	selector types.ChainSelector,
	source types.ChainMetadata,
	action types.TimelockAction,
	resolver RoleAddressResolver,
) (types.ChainMetadata, error) {
	role, err := TimelockRoleFromAction(action)
	if err != nil {
		return types.ChainMetadata{}, err
	}
	if resolver == nil {
		return types.ChainMetadata{}, fmt.Errorf("role address resolver is required to resolve the %s MCM address", role)
	}

	address, err := resolver.ResolveRoleAddress(ctx, RoleAddressRequest{
		Selector:         selector,
		Role:             role,
		SourceMCMAddress: source.MCMAddress,
	})
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("failed to resolve %s MCM address for chain %d: %w", role, selector, err)
	}
	if address == "" {
		return types.ChainMetadata{}, fmt.Errorf("empty %s MCM address resolved for chain %d", role, selector)
	}

	derived := source
	derived.MCMAddress = address

	return derived, nil
}
