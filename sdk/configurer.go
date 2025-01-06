package sdk

import (
	"github.com/smartcontractkit/mcms/types"
)

type Configurer interface {
	SetConfig(mcmID types.ContractID, cfg *types.Config, clearRoot bool) (string, error)
}
