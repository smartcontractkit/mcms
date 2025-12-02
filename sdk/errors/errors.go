package sdkerrors

import (
	"fmt"

	"github.com/smartcontractkit/mcms/types"
)

type InvalidChainIDError struct {
	ReceivedChainID types.ChainSelector
}

func (e *InvalidChainIDError) Error() string {
	return fmt.Sprintf("invalid chain ID: %v", e.ReceivedChainID)
}

func NewInvalidChainIDError(receivedChainID types.ChainSelector) *InvalidChainIDError {
	return &InvalidChainIDError{ReceivedChainID: receivedChainID}
}

type TooManySignersError struct {
	NumSigners uint64
}

func (e *TooManySignersError) Error() string {
	return fmt.Sprintf("too many signers: %v max number is 255", e.NumSigners)
}

func NewTooManySignersError(numSigners uint64) *TooManySignersError {
	return &TooManySignersError{NumSigners: numSigners}
}

// Error for an invalid timelock operation.
type InvalidTimelockOperationError struct {
	ReceivedTimelockOperation string
}

// Error returns the error message.
func (e *InvalidTimelockOperationError) Error() string {
	return "invalid timelock operation: " + e.ReceivedTimelockOperation
}

func NewInvalidTimelockOperationError(op string) *InvalidTimelockOperationError {
	return &InvalidTimelockOperationError{ReceivedTimelockOperation: op}
}
