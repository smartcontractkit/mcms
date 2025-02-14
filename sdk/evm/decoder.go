package evm

import (
	"encoding/json"
	"strings"

	geth_abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/mcms/internal/utils/abi"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type DecodedOperation struct {
	FunctionName string
	InputArgs    []any
	InputKeys    geth_abi.Arguments
}

var _ sdk.DecodedOperation = &DecodedOperation{}

func (d *DecodedOperation) MethodName() string {
	return d.FunctionName
}

func (d *DecodedOperation) Args() []any {
	return d.InputArgs
}

func (d *DecodedOperation) String() (string, string, error) {
	// Create a human readable representation of the decoded operation
	// by displaying a map of input keys to input values
	// e.g. {"key1": "value1", "key2": "value2"}
	inputMap := make(map[string]any)
	for i, key := range d.InputKeys {
		inputMap[key.Name] = d.InputArgs[i]
	}

	byteMap, err := json.Marshal(inputMap)
	if err != nil {
		return "", "", err
	}

	return d.FunctionName, string(byteMap), nil
}

type Decoder struct {
}

var _ sdk.Decoder = &Decoder{}

func (d *Decoder) Decode(op types.Operation, abi string) (sdk.DecodedOperation, error) {
	return ParseFunctionCall(abi, op.Transaction.Data)
}

// ParseFunctionCall parses a full data payload (with function selector at the front of it) and a full contract ABI
// into a function name and an array of inputs.
func ParseFunctionCall(fullAbi string, data []byte) (*DecodedOperation, error) {
	// Parse the ABI
	parsedAbi, err := geth_abi.JSON(strings.NewReader(fullAbi))
	if err != nil {
		return &DecodedOperation{}, err
	}

	// Extract the method from the data
	method, err := parsedAbi.MethodById(data[:4])
	if err != nil {
		return &DecodedOperation{}, err
	}

	// Marshal the method's inputs
	byteInputs, err := json.Marshal(method.Inputs)
	if err != nil {
		return &DecodedOperation{}, err
	}

	// Decode the data using the method's input types
	inputs, err := abi.ABIDecode(string(byteInputs), data[4:])
	if err != nil {
		return &DecodedOperation{}, err
	}

	return &DecodedOperation{
		FunctionName: method.Name,
		InputKeys:    method.Inputs,
		InputArgs:    inputs,
	}, nil
}
