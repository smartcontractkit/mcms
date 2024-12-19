package evm

import (
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

const maxUint8Value = 255

type ConfigTransformer struct{}

func NewConfigTransformer() *ConfigTransformer {
	return &ConfigTransformer{}
}

// ToConfig converts an EVM ManyChainMultiSigConfig to a chain-agnostic types.Config
func (e *ConfigTransformer) ToConfig(
	bindConfig bindings.ManyChainMultiSigConfig,
) (*types.Config, error) {
	panic("implement me")
}

// ToChainConfig converts a chain-agnostic types.Config to an EVM ManyChainMultiSigConfig
func (e *ConfigTransformer) ToChainConfig(
	cfg types.Config,
) (bindings.ManyChainMultiSigConfig, error) {
	panic("implement me")
}
