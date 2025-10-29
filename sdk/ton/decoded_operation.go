package ton

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
)

type decodedOperation struct {
	contractType string
	msgType      string
	msgOpcode    uint64

	// Message data
	msgDecoded any // normalized and decoded cell structure
	inputKeys  []string
	inputArgs  []any
}

var _ sdk.DecodedOperation = &decodedOperation{}

func NewDecodedOperation(contractType string, msgType string, msgOpcode uint64, msgDecoded any, inputKeys []string, inputArgs []any) (sdk.DecodedOperation, error) {
	if len(inputKeys) != len(inputArgs) {
		return nil, fmt.Errorf("input keys and input args must have the same length")
	}

	return &decodedOperation{contractType, msgType, msgOpcode, msgDecoded, inputKeys, inputArgs}, nil
}

func (o *decodedOperation) MethodName() string {
	return fmt.Sprintf("%s::%s(0x%x)", o.contractType, o.msgType, o.msgOpcode)
}

func (o *decodedOperation) Keys() []string {
	return o.inputKeys
}

func (o *decodedOperation) Args() []any {
	return o.inputArgs
}

func (o *decodedOperation) String() (string, string, error) {
	// Create a human readable representation of the decoded operation
	// by displaying a map of input keys to input values
	// e.g. {"key1": "value1", "key2": "value2"}

	// Notice: cell is an encoded tree structure, where args can be nested so we print
	// out the full decoded structure here, but we only return the first layer of keys
	// and args via Keys() and Args() respective funcs.

	byteMap, err := json.MarshalIndent(o.msgDecoded, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to JSON marshal the decoded op: %w", err)
	}

	return o.MethodName(), string(byteMap), nil
}
