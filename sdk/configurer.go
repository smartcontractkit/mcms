package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

type Configurer interface {
	SetConfig(mcmAddr string, cfg *types.Config, clearRoot bool) (string, error)
}
