package stellar

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/strkey"

	"github.com/smartcontractkit/mcms/types"
)

type contractKind uint8

const (
	contractKindUnknown contractKind = iota
	contractKindMCMS
	contractKindTimelock
)

var mcmsErrorNames = map[uint32]string{
	12: "OutOfBoundsGroup",
	45: "CallReverted",
}

var timelockErrorNames = map[uint32]string{
	32: "CallReverted",
}

const maxScValErrorDecodeDepth = 32

type ContractError struct {
	ContractID string
	Code       uint32
	Name       string
}

// ExecutionError contains the errors reported by contracts during execution.
//
// ErrorChain is ordered from the outer contract error to the root cause.
type ExecutionError struct {
	Transaction *types.Transaction
	ErrorChain  []ContractError
	OriginalErr error
}

func (e *ExecutionError) Error() string {
	if root := e.RootCause(); root != nil {
		name := root.Name
		if name == "" {
			name = fmt.Sprintf("UnknownContractError(%d)", root.Code)
		}

		return fmt.Sprintf("Stellar transaction execution failed: %s", name)
	}

	return fmt.Sprintf(
		"Stellar transaction execution failed: %v",
		e.OriginalErr,
	)
}

func (e *ExecutionError) Unwrap() error {
	return e.OriginalErr
}

// OuterError returns the error reported by the contract that the SDK called
// directly.
func (e *ExecutionError) OuterError() *ContractError {
	if len(e.ErrorChain) == 0 {
		return nil
	}

	return &e.ErrorChain[0]
}

// RootCause returns the original contract error that caused execution to fail.
func (e *ExecutionError) RootCause() *ContractError {
	if len(e.ErrorChain) == 0 {
		return nil
	}

	return &e.ErrorChain[len(e.ErrorChain)-1]
}

func newExecutionError(tx *types.Transaction, err error, knownContracts map[string]contractKind) error {
	if err == nil {
		return nil
	}

	errorChain, decodeErr := DecodeSimulationErrorChain(err)
	if decodeErr != nil && len(errorChain) == 0 {
		return &ExecutionError{
			Transaction: tx,
			OriginalErr: err,
		}
	}

	for i := range errorChain {
		kind := knownContracts[canonicalContractID(errorChain[i].ContractID)]

		errorChain[i].Name = resolveContractErrorName(
			kind,
			errorChain[i].Code,
		)
	}

	// SimulateTransaction.EventsXDR is chronological. The original contract
	// error is first, and the outer contract error is last.
	//
	// Normalize ErrorChain so that the outer contract error is first and the
	// root cause is last.
	reverseErrorChain(errorChain)

	return &ExecutionError{
		Transaction: tx,
		ErrorChain:  errorChain,
		OriginalErr: err,
	}
}

func reverseErrorChain(errorChain []ContractError) {
	for left, right := 0, len(errorChain)-1; left < right; {
		errorChain[left], errorChain[right] =
			errorChain[right], errorChain[left]

		left++
		right--
	}
}

func resolveContractErrorName(
	kind contractKind,
	code uint32,
) string {
	switch kind {
	case contractKindMCMS:
		if name, ok := mcmsErrorNames[code]; ok {
			return name
		}
	case contractKindTimelock:
		if name, ok := timelockErrorNames[code]; ok {
			return name
		}
	}

	// An error can come from a nested contract that the executor does not
	// know. Resolve the name only when the code has one unambiguous meaning.
	mcmsName, inMCMS := mcmsErrorNames[code]
	timelockName, inTimelock := timelockErrorNames[code]

	switch {
	case inMCMS && !inTimelock:
		return mcmsName
	case inTimelock && !inMCMS:
		return timelockName
	default:
		return fmt.Sprintf("UnknownContractError(%d)", code)
	}
}

func canonicalContractID(contractID string) string {
	raw, err := parseContractID(contractID)
	if err != nil {
		return contractID
	}

	encoded, err := strkey.Encode(
		strkey.VersionByteContract,
		raw[:],
	)
	if err != nil {
		return contractID
	}

	return encoded
}
