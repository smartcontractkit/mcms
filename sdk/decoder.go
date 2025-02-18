package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

// Decoder decodes the transaction data of chain operations.
//
// This is only required if the chain supports decoding.
type Decoder interface {
	// Decode decodes the transaction data of a chain operation.
	//
	// contractInterfaces is the ABI of the contract that the operation is interacting with.
	Decode(op types.Transaction, contractInterfaces string) (DecodedOperation, error)
}
