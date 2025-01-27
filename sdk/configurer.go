package sdk

import (
	"context"

	"github.com/smartcontractkit/mcms/types"
)

type Configurer interface {
	SetConfig(ctx context.Context, mcmAddr string, cfg *types.Config, clearRoot bool) (types.NativeTransaction, error)
}
