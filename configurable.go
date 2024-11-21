package mcms

import (
	"errors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type Configurable struct {
	configurer map[types.ChainSelector]sdk.Configurer
}

func NewConfigurable(
	configurer map[types.ChainSelector]sdk.Configurer,
) *Configurable {
	return &Configurable{
		configurer: configurer,
	}
}

// SetConfig sets the configuration delegating to the underlying Configurer.
func (c *Configurable) SetConfig(
	csel types.ChainSelector, cfg *types.Config, clearRoot bool,
) (string, error) {
	cfger, ok := c.configurer[csel]
	if !ok {
		return "", errors.New("wtf")
	}

	return cfger.SetConfig(cfg, clearRoot)
}
