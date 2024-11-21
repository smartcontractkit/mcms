package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

type Configurer interface {
	SetConfig(cfg *types.Config, clearRoot bool) (string, error)
}
