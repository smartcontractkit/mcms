package ton

import (
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
)

type DecodedOperation struct {
	ContractType string
	MsgType      string
	MsgOpcode    uint64

	// Message data
	MsgDecoded any // normalized and decoded cell structure
	InputKeys  []string
	InputArgs  []any
}

var _ sdk.DecodedOperation = &DecodedOperation{}

func NewDecodedOperation(contractType string, msgType string, msgOpcode uint64, msgDecoded any, inputKeys []string, inputArgs []any) (sdk.DecodedOperation, error) {
	if len(inputKeys) != len(inputArgs) {
		return nil, fmt.Errorf("input keys and input args must have the same length")
	}

	return &DecodedOperation{contractType, msgType, msgOpcode, msgDecoded, inputKeys, inputArgs}, nil
}

func (o *DecodedOperation) MethodName() string {
	return fmt.Sprintf("%s::%s(0x%x)", o.ContractType, o.MsgType, o.MsgOpcode)
}

func (o *DecodedOperation) Keys() []string {
	return o.InputKeys
}

func (o *DecodedOperation) Args() []any {
	return o.InputArgs
}

func (o *DecodedOperation) String() (string, string, error) {
	// Create a human readable representation of the decoded operation
	// by displaying a map of input keys to input values
	// e.g. {"key1": "value1", "key2": "value2"}

	// Notice: cell is an encoded tree structure, where args can be nested so we print
	// out the full decoded structure here, but we only return the first layer of keys
	// and args via Keys() and Args() respective funcs.

	byteMap, err := json.MarshalIndent(o.MsgDecoded, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to JSON marshal the decoded op: %w", err)
	}

	return o.MethodName(), string(byteMap), nil
}
