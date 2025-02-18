package evm

import (
	"encoding/json"
	"strings"

	geth_abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/mcms/internal/utils/abi"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type Decoder struct{}

var _ sdk.Decoder = &Decoder{}

func (d *Decoder) Decode(op types.Operation, contractInterfaces string) (sdk.DecodedOperation, error) {
	return ParseFunctionCall(contractInterfaces, op.Transaction.Data)
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

	methodKeys := make([]string, len(method.Inputs))
	for i, input := range method.Inputs {
		methodKeys[i] = input.Name
	}

	return &DecodedOperation{
		FunctionName: method.Name,
		InputKeys:    methodKeys,
		InputArgs:    inputs,
	}, nil
}
