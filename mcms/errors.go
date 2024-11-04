package mcms

import (
	"fmt"

	"github.com/smartcontractkit/mcms/types"
)

// ChainMetadataNotFoundError is returned when the chain metadata for a chain is not found in a
// proposal.
type ChainMetadataNotFoundError struct {
	ChainSelector types.ChainSelector
}

// NewChainMetadataNotFoundError creates a new ChainMetadataNotFoundError.
func NewChainMetadataNotFoundError(sel types.ChainSelector) *ChainMetadataNotFoundError {
	return &ChainMetadataNotFoundError{ChainSelector: sel}
}

// Error implements the error interface.
func (e *ChainMetadataNotFoundError) Error() string {
	return fmt.Sprintf("missing metadata for chain %d", e.ChainSelector)
}

// InconsistentConfigsError is returned when the configs for two chains are not equal to each
// other.
type InconsistentConfigsError struct {
	ChainSelectorA types.ChainSelector
	ChainSelectorB types.ChainSelector
}

// NewInconsistentConfigsError creates a new InconsistentConfigsError.
func NewInconsistentConfigsError(selA, selB types.ChainSelector) *InconsistentConfigsError {
	return &InconsistentConfigsError{ChainSelectorA: selA, ChainSelectorB: selB}
}

// Error implements the error interface.
func (e *InconsistentConfigsError) Error() string {
	return fmt.Sprintf("inconsistent configs for chains %d and %d", e.ChainSelectorA, e.ChainSelectorB)
}

// QuorumNotReachedError is returned when the quorum has not been reach as defined in a chain's
// MCM contract configuration.
type QuorumNotReachedError struct {
	ChainSelector types.ChainSelector
}

// NewQuorumNotReachedError creates a new QuorumNotReachedError.
func NewQuorumNotReachedError(sel types.ChainSelector) *QuorumNotReachedError {
	return &QuorumNotReachedError{ChainSelector: sel}
}

// Error implements the error interface.
func (e *QuorumNotReachedError) Error() string {
	return fmt.Sprintf("quorum not reached for chain %d", e.ChainSelector)
}
