package timelock

import (
	"github.com/smartcontractkit/mcms/types"
)

type BatchChainOperation struct {
	ChainSelector types.ChainSelector `json:"chainIdentifier"`
	Batch         []types.Operation   `json:"batch"`
}
