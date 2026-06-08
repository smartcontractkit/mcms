package canton

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
)

type DecodedOperation struct {
	ContractType string
	FunctionName string

	InputKeys []string
	InputArgs []any
}

var _ sdk.DecodedOperation = &DecodedOperation{}

func NewDecodedOperation(contractType, functionName string, inputKeys []string, inputArgs []any) (*DecodedOperation, error) {
	if len(inputKeys) != len(inputArgs) {
		return nil, errors.New("input keys and input args must have the same length")
	}

	return &DecodedOperation{
		ContractType: contractType,
		FunctionName: functionName,
		InputKeys:    inputKeys,
		InputArgs:    inputArgs,
	}, nil
}

func (d DecodedOperation) MethodName() string {
	if d.ContractType == "" {
		return d.FunctionName
	}

	return fmt.Sprintf("%s::%s", d.ContractType, d.FunctionName)
}

func (d DecodedOperation) Keys() []string {
	return d.InputKeys
}

func (d DecodedOperation) Args() []any {
	return d.InputArgs
}

func (d DecodedOperation) String() (string, string, error) {
	inputMap := make(map[string]any, len(d.InputKeys))
	for i, key := range d.InputKeys {
		inputMap[key] = d.InputArgs[i]
	}

	byteMap, err := json.MarshalIndent(inputMap, "", "  ")
	if err != nil {
		return "", "", err
	}

	return d.FunctionName, string(byteMap), nil
}
