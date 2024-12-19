package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

type Configurer[T any] interface {
	SetConfig(mcmAddr T, cfg *types.Config, clearRoot bool) (string, error)
}
