package mcms

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// InvalidProposalKindError is returned when an invalid proposal kind is provided.
type InvalidProposalKindError struct {
	ProvidedKind types.ProposalKind
	AcceptedKind types.ProposalKind
}

func (e *InvalidProposalKindError) Error() string {
	return fmt.Sprintf("invalid proposal kind: %s, value accepted is %s", e.ProvidedKind, e.AcceptedKind)
}
func NewInvalidProposalKindError(provided, accepted types.ProposalKind) *InvalidProposalKindError {
	return &InvalidProposalKindError{ProvidedKind: provided, AcceptedKind: accepted}
}

// EncoderNotFoundError is returned when an encoder is not found for a chain in a proposal.
type EncoderNotFoundError struct {
	ChainSelector types.ChainSelector
}

// NewEncoderNotFoundError creates a new EncoderNotFoundError.
func NewEncoderNotFoundError(sel types.ChainSelector) *EncoderNotFoundError {
	return &EncoderNotFoundError{ChainSelector: sel}
}

func (e *EncoderNotFoundError) Error() string {
	return fmt.Sprintf("encoder not provided for chain selector %d", e.ChainSelector)
}

// ChainMetadataNotFoundError is returned when the chain metadata for a chain is not found in a
// proposal.
type ChainMetadataNotFoundError struct {
	ChainSelector types.ChainSelector
}

// NewChainMetadataNotFoundError creates a new ChainMetadataNotFoundError.
func NewChainMetadataNotFoundError(sel types.ChainSelector) *ChainMetadataNotFoundError {
	return &ChainMetadataNotFoundError{ChainSelector: sel}
}

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

func (e *QuorumNotReachedError) Error() string {
	return fmt.Sprintf("quorum not reached for chain %d", e.ChainSelector)
}

// InvalidDelayError is the error for when the received delay for Timelock is invalid.
type InvalidDelayError struct {
	ReceivedDelay string
}

// Error returns the error message.
func (e *InvalidDelayError) Error() string {
	return fmt.Sprintf("invalid delay: %s", e.ReceivedDelay)
}

func NewInvalidDelayError(receivedDelay string) *InvalidDelayError {
	return &InvalidDelayError{ReceivedDelay: receivedDelay}
}

type InvalidValidUntilError struct {
	ReceivedValidUntil uint32
}

func (e *InvalidValidUntilError) Error() string {
	return fmt.Sprintf("invalid valid until: %v", e.ReceivedValidUntil)
}

func NewInvalidValidUntilError(receivedValidUntil uint32) *InvalidValidUntilError {
	return &InvalidValidUntilError{ReceivedValidUntil: receivedValidUntil}
}

var ErrNoTransactionsInBatch = errors.New("no transactions in batch")

type InvalidSignatureError struct {
	RecoveredAddress common.Address
}

func (e *InvalidSignatureError) Error() string {
	return fmt.Sprintf("invalid signature: received signature for address %s is not a valid signer in the MCMS proposal", e.RecoveredAddress)
}

func NewInvalidSignatureError(recoveredAddress common.Address) *InvalidSignatureError {
	return &InvalidSignatureError{RecoveredAddress: recoveredAddress}
}
