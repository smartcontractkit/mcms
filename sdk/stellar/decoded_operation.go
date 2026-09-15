package stellar

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.DecodedOperation = (*DecodedOperation)(nil)

// DecodedOperation is the sdk.DecodedOperation implementation for Stellar.
//
// Argument keys are positional placeholders (arg0, arg1, ...) except for the
// RBACTimelock entrypoints, whose parameter names are recorded in
// timelockArgNames. Soroban contract specs carry parameter names, but the
// generated chainlink-stellar bindings emit only client.go + types.go per
// contract — no JSON spec and no function-info registry — and fetching the
// spec would need an RPC round-trip the offline sdk.Decoder interface does
// not allow. Target contract, function name, and argument values all still
// render, which is what removes the blind-signing risk; named args are
// readability, not correctness.
type DecodedOperation struct {
	ContractType string
	Function     string

	// rawArgs holds the decoded Soroban arguments. The field cannot be named
	// Args: the sdk.DecodedOperation interface requires an Args() method.
	rawArgs []xdr.ScVal
}

// timelockArgNames gives the parameter names of the RBACTimelock
// entrypoints, taken from chainlink-stellar contracts/timelock/src/lib.rs.
// Naming them is not cosmetic: timelock conversion expands a batch's "calls"
// argument, so an outer call whose batch argument is named argN would keep
// the raw encoded batch alongside the expanded entry instead of replacing
// it. The signatures are fixed knowledge, already hard-coded by the timelock
// batch checkers. A known function whose argument count does not match its
// recorded signature falls back to positional names, so a future contract
// revision mislabels nothing.
// Recorded timelock parameter names.
const (
	timelockArgCaller      = "caller"
	timelockArgCalls       = "calls"
	timelockArgPredecessor = "predecessor"
	timelockArgSalt        = "salt"
	timelockArgDelay       = "delay"
	timelockArgID          = "id"
)

var timelockArgNames = map[string][]string{
	"schedule_batch":         {timelockArgCaller, timelockArgCalls, timelockArgPredecessor, timelockArgSalt, timelockArgDelay},
	"bypasser_execute_batch": {timelockArgCaller, timelockArgCalls},
	"execute_batch":          {timelockArgCalls, timelockArgPredecessor, timelockArgSalt},
	"cancel":                 {timelockArgCaller, timelockArgID},
}

// argNames returns the input-argument names for count arguments of function:
// the recorded parameter names for the timelock entrypoints, positional
// placeholders otherwise.
func argNames(function string, count int) []string {
	if known, ok := timelockArgNames[function]; ok && len(known) == count {
		return known
	}

	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("arg%d", i)
	}

	return names
}

// NewDecodedOperation builds a DecodedOperation from a decoded Soroban
// invocation. len(Keys()) always equals len(Args()), which the analyzer's
// NamedField zip depends on.
func NewDecodedOperation(contractType, function string, args []xdr.ScVal) (*DecodedOperation, error) {
	if function == "" {
		return nil, errors.New("stellar decoded operation has no function name")
	}

	return &DecodedOperation{
		ContractType: contractType,
		Function:     function,
		rawArgs:      args,
	}, nil
}

// MethodName returns the invoked function, prefixed with the contract type
// when one is known (mirroring the other families' namespaced method names).
func (o *DecodedOperation) MethodName() string {
	if o.ContractType == "" {
		return o.Function
	}

	return fmt.Sprintf("%s::%s", o.ContractType, o.Function)
}

// Keys returns the input-argument names: the recorded parameter names for
// the timelock entrypoints, positional placeholders otherwise.
func (o *DecodedOperation) Keys() []string {
	return argNames(o.Function, len(o.rawArgs))
}

// Args returns display-friendly representations of the input arguments.
func (o *DecodedOperation) Args() []any {
	args := make([]any, len(o.rawArgs))
	for i, raw := range o.rawArgs {
		args[i] = ScValToDisplay(raw)
	}

	return args
}

// String returns a human-readable representation of the decoded operation:
// the function name and a JSON object mapping each argument name to its
// display value.
func (o *DecodedOperation) String() (string, string, error) {
	names := o.Keys()

	inputMap := make(map[string]any, len(o.rawArgs))
	for i, raw := range o.rawArgs {
		inputMap[names[i]] = ScValToDisplay(raw)
	}

	encoded, err := json.MarshalIndent(inputMap, "", "  ")
	if err != nil {
		return "", "", err
	}

	return o.Function, string(encoded), nil
}
