package sdk

import (
	"github.com/smartcontractkit/mcms/internal/core/config"
)

// R in this case is the chain-specific struct thats the set of inputs
// required to make a SetConfig call
type Configurator[R any] interface {
	// Returns a string with the transaction hash
	SetConfigInputs(contract string, cfg config.Config) (R, error)

	// Maps to the chain-agnostic config
	ToConfig(onchainConfig R) (*config.Config, error)
}
