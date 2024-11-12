package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

// Decoder decodes the transaction data of chain operations.
//
// This is only required if the chain supports decoding.
type Decoder interface {
	Decode(op types.Operation, abi string) (methodName string, args string, err error)
}
