package mcms

import (
	"github.com/smartcontractkit/mcms/types"
)

// Defined on a per chain level
type Decoder interface {
	// Returns: (MethodName, Args, error)
	Decode(operation types.ChainOperation, abiStr string) (string, string, error)
}
