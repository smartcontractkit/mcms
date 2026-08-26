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

type ExecutionError struct {
	Transaction *types.Transaction
	Frames      []ContractError
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

func (e *ExecutionError) OuterError() *ContractError {
	if len(e.Frames) == 0 {
		return nil
	}

	return &e.Frames[0]
}

func (e *ExecutionError) RootCause() *ContractError {
	if len(e.Frames) == 0 {
		return nil
	}

	return &e.Frames[len(e.Frames)-1]
}

func newExecutionError(tx *types.Transaction, err error, knownContracts map[string]contractKind) error {
	if err == nil {
		return nil
	}

	frames, decodeErr := DecodeSimulationErrorFrames(err)
	if decodeErr != nil && len(frames) == 0 {
		return &ExecutionError{
			Transaction: tx,
			OriginalErr: err,
		}
	}

	for i := range frames {
		kind := knownContracts[canonicalContractID(frames[i].ContractID)]

		frames[i].Name = resolveContractErrorName(
			kind,
			frames[i].Code,
		)
	}

	// SimulateTransaction.EventsXDR is chronological:
	// root cause first, outer wrapper last.
	//
	// Normalize ExecutionError.Frames to:
	// outer wrapper first, root cause last.
	reverseContractErrors(frames)

	return &ExecutionError{
		Transaction: tx,
		Frames:      frames,
		OriginalErr: err,
	}
}

func reverseContractErrors(frames []ContractError) {
	for left, right := 0, len(frames)-1; left < right; {
		frames[left], frames[right] =
			frames[right], frames[left]

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

	// A diagnostic frame may come from a nested contract not explicitly
	// registered by the executor. Resolve only codes that are unambiguous
	// between the MCMS and timelock contracts.
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
