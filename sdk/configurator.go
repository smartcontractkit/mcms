package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

// R in this case is the chain-specific struct thats the set of inputs
// required to make a SetConfig call
type Configurator[R any] interface {
	// Returns a string with the transaction hash
	SetConfigInputs(contract string, cfg types.Config) (R, error)

	// Maps to the chain-agnostic config
	ToConfig(onchainConfig R) (*types.Config, error)
}
