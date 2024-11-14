package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

// ConfigTransformer is the interface used to create the configuration of an MCMS contract.
// R in this case is the chain-specific struct that is used to configure the contract.
// the interface allows conversion between the chain-specific struct and the chain-agnostic.
type ConfigTransformer[R any] interface {
	// ToChainConfig converts the chain agnostic config to the chain-specific config
	ToChainConfig(contract string, cfg types.Config) (R, error)

	// ToConfig Maps the chain-specific config to the chain-agnostic config
	ToConfig(onchainConfig R) (*types.Config, error)
}
