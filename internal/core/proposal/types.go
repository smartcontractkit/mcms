package proposal

import (
	"github.com/smartcontractkit/mcms/types"
)

type Executable interface {
	SetRoot(chainSelector types.ChainSelector) (string, error)
	Execute(index int) (string, error)
}
