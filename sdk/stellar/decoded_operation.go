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
// Positional keys are a known limitation: Soroban contract specs carry
// parameter names, but the generated chainlink-stellar bindings emit only
// client.go + types.go per contract — no JSON spec and no function-info
// registry — and fetching the spec would need an RPC round-trip the offline
// sdk.Decoder interface does not allow. Target contract, function name, and
// argument values all still render, which is what removes the blind-signing
// risk; named args are readability, not correctness.
type DecodedOperation struct {
	ContractType string
	Function     string

	// rawArgs holds the decoded Soroban arguments. The field cannot be named
	// Args: the sdk.DecodedOperation interface requires an Args() method.
	rawArgs []xdr.ScVal
}

// NewDecodedOperation builds a DecodedOperation from a decoded Soroban
// invocation. Keys are synthesized as positional placeholders (arg0, arg1,
// ...) because Soroban carries no parameter names; len(Keys()) always equals
// len(Args()), which the analyzer's NamedField zip depends on.
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

// Keys returns the input-argument names: positional placeholders, since
// Soroban invocations carry no parameter names.
func (o *DecodedOperation) Keys() []string {
	keys := make([]string, len(o.rawArgs))
	for i := range o.rawArgs {
		keys[i] = fmt.Sprintf("arg%d", i)
	}

	return keys
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
// the function name and a JSON object mapping each argument placeholder to
// its display value.
func (o *DecodedOperation) String() (string, string, error) {
	inputMap := make(map[string]any, len(o.rawArgs))
	for i, raw := range o.rawArgs {
		inputMap[fmt.Sprintf("arg%d", i)] = ScValToDisplay(raw)
	}

	encoded, err := json.MarshalIndent(inputMap, "", "  ")
	if err != nil {
		return "", "", err
	}

	return o.Function, string(encoded), nil
}
