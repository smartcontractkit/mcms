package core

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var ErrEmptyDescription = errors.New("invalid empty description")
var ErrNoChainMetadata = errors.New("no chain metadata")
var ErrNoTransactions = errors.New("no transactions")
var ErrNoTransactionsInBatch = errors.New("no transactions in batch")

// InvalidChainIDError is the error for an invalid chain ID.
type InvalidChainIDError struct {
	ReceivedChainID uint64
}

// Error returns the error message.
func (e *InvalidChainIDError) Error() string {
	return fmt.Sprintf("invalid chain ID: %v", e.ReceivedChainID)
}

// InvalidDelayError is the error for when the received min delay is invalid.
type InvalidDelayError struct {
	ReceivedDelay string
}

// Error returns the error message.
func (e *InvalidDelayError) Error() string {
	return fmt.Sprintf("invalid delay: %s", e.ReceivedDelay)
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

type InvalidSignatureError struct {
	RecoveredAddress common.Address
}

func (e *InvalidSignatureError) Error() string {
	return fmt.Sprintf("invalid signature: received signature for address %s is not a signer on the MCMS contract", e.RecoveredAddress)
}

type InvalidMCMSConfigError struct {
	Reason string
}

func (e *InvalidMCMSConfigError) Error() string {
	return fmt.Sprintf("invalid MCMS config: %s", e.Reason)
}

type TooManySignersError struct {
	NumSigners uint64
}

func (e *TooManySignersError) Error() string {
	return fmt.Sprintf("too many signers: %v max number is 255", e.NumSigners)
}
