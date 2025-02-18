package evm

import (
	"encoding/json"

	geth_abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/mcms/sdk"
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

	byteMap, err := json.MarshalIndent(inputMap, "", "  ")
	if err != nil {
		return "", "", err
	}

	return d.FunctionName, string(byteMap), nil
}
