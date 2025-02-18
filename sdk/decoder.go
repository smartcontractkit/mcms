package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

type DecodedOperation interface {
	MethodName() string
	Args() []any

	// String returns a human readable representation of the decoded operation.
	//
	// The first return value is the method name.
	// The second return value is a string representation of the input arguments.
	// The third return value is an error if there was an issue generating the string.
	String() (string, string, error)
}

// Decoder decodes the transaction data of chain operations.
//
// This is only required if the chain supports decoding.
type Decoder interface {
	// Decode decodes the transaction data of a chain operation.
	//
	// contractInterfaces is the ABI of the contract that the operation is interacting with.
	Decode(op types.Operation, contractInterfaces string) (DecodedOperation, error)
}
