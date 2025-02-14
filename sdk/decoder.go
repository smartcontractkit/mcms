package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

type DecodedOperation interface {
	MethodName() string
	Args() []any    // TODO: this maybe should be a generic type
	String() string // human readable representation
}

// Decoder decodes the transaction data of chain operations.
//
// This is only required if the chain supports decoding.
type Decoder interface {
	Decode(op types.Operation, abi string) (DecodedOperation, error)
}
