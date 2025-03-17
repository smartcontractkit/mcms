package evm

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
)

type DecodedOperation struct {
	FunctionName string
	InputArgs    []any
	InputKeys    []string
}

var _ sdk.DecodedOperation = &DecodedOperation{}

func NewDecodedOperation(functionName string, inputKeys []string, inputArgs []any) (*DecodedOperation, error) {
	if len(inputKeys) != len(inputArgs) {
		return nil, fmt.Errorf("input keys and input args must have the same length")
	}

	return &DecodedOperation{
		FunctionName: functionName,
		InputKeys:    inputKeys,
		InputArgs:    inputArgs,
	}, nil
}

func (d *DecodedOperation) MethodName() string {
	return d.FunctionName
}

func (d *DecodedOperation) Keys() []string {
	return d.InputKeys
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
		inputMap[key] = d.InputArgs[i]
	}

	byteMap, err := json.MarshalIndent(inputMap, "", "  ")
	if err != nil {
		return "", "", err
	}

	return d.FunctionName, string(byteMap), nil
}
