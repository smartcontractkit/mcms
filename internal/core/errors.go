package core

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

type UnknownChainSelectorFamilyError struct {
	ChainSelector uint64
	ChainFamily   string
}

var SupportedChainSelectorFamilies = []string{
	chain_selectors.FamilyEVM,
	chain_selectors.FamilySolana,
}

func (e UnknownChainSelectorFamilyError) Error() string {
	return fmt.Sprintf("unknown chain selector family: %d with family %s. Supported families are %v", e.ChainSelector, e.ChainFamily, SupportedChainSelectorFamilies)
}

func NewUnknownChainSelectorFamilyError(selector uint64, family string) *UnknownChainSelectorFamilyError {
	return &UnknownChainSelectorFamilyError{
		ChainSelector: selector,
		ChainFamily:   family,
	}
}

// InvalidChainIDError is the error for an invalid chain ID.
type InvalidChainIDError struct {
	ReceivedChainID uint64
}

// Error returns the error message.
func (e *InvalidChainIDError) Error() string {
	return fmt.Sprintf("invalid chain ID: %v", e.ReceivedChainID)
}

type InvalidDescriptionError struct {
	ReceivedDescription string
}

func (e *InvalidDescriptionError) Error() string {
	return fmt.Sprint("invalid description: ", e.ReceivedDescription)
}

// InvalidMinDelayError is the error for when the received min delay is invalid.
type InvalidMinDelayError struct {
	ReceivedMinDelay string
}

// Error returns the error message.
func (e *InvalidMinDelayError) Error() string {
	return fmt.Sprintf("invalid min delay: %s", e.ReceivedMinDelay)
}

// InvalidProposalTypeError is used when an invalid proposal type is received.
type InvalidProposalTypeError struct {
	ReceivedProposalType string
}

func (e *InvalidProposalTypeError) Error() string {
	return fmt.Sprintf("invalid proposal type: %s", e.ReceivedProposalType)
}

// InvalidTimelockOperationError is the error for an invalid timelock operation.
type InvalidTimelockOperationError struct {
	ReceivedTimelockOperation string
}

// Error returns the error message.
func (e *InvalidTimelockOperationError) Error() string {
	return fmt.Sprintf("invalid timelock operation: %s", e.ReceivedTimelockOperation)
}

type InvalidValidUntilError struct {
	ReceivedValidUntil uint32
}

func (e *InvalidValidUntilError) Error() string {
	return fmt.Sprintf("invalid valid until: %v", e.ReceivedValidUntil)
}

type InvalidVersionError struct {
	ReceivedVersion string
}

func (e *InvalidVersionError) Error() string {
	return fmt.Sprintf("invalid version: %s", e.ReceivedVersion)
}

// MissingChainDetailsError is the error for missing chain metadata.
type MissingChainDetailsError struct {
	Parameter       string
	ChainIdentifier uint64
}

// Error returns the error message.
func (e *MissingChainDetailsError) Error() string {
	return fmt.Sprintf("missing %s for chain %v", e.Parameter, e.ChainIdentifier)
}

// MissingChainClientError is the error for missing chain client.
type MissingChainClientError struct {
	ChainIdentifier uint64
}

// Error returns the error message.
func (e *MissingChainClientError) Error() string {
	return fmt.Sprintf("missing chain client for chain %v", e.ChainIdentifier)
}

type NoChainMetadataError struct {
}

func (e *NoChainMetadataError) Error() string {
	return "no chain metadata"
}

type NoTransactionsError struct {
}

func (e *NoTransactionsError) Error() string {
	return "no transactions"
}

type InvalidSignatureError struct {
	ChainIdentifier  uint64
	MCMSAddress      common.Address
	RecoveredAddress common.Address
}

func (e *InvalidSignatureError) Error() string {
	return fmt.Sprintf("invalid signature: received signature for address %s is not a signer on MCMS %s on chain %v", e.RecoveredAddress, e.MCMSAddress, e.ChainIdentifier)
}

type InvalidMCMSConfigError struct {
	Reason string
}

func (e *InvalidMCMSConfigError) Error() string {
	return fmt.Sprintf("invalid MCMS config: %s", e.Reason)
}

type QuorumNotMetError struct {
	ChainIdentifier uint64
}

func (e *QuorumNotMetError) Error() string {
	return fmt.Sprintf("quorum not met for chain %v", e.ChainIdentifier)
}

type InconsistentConfigsError struct {
	ChainIdentifierA uint64
	ChainIdentifierB uint64
}

func (e *InconsistentConfigsError) Error() string {
	return fmt.Sprintf("inconsistent configs for chains %v and %v", e.ChainIdentifierA, e.ChainIdentifierB)
}

type TooManySignersError struct {
	NumSigners uint64
}

func (e *TooManySignersError) Error() string {
	return fmt.Sprintf("too many signers: %v max number is 255", e.NumSigners)
}
