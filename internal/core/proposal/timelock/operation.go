package timelock

import "github.com/smartcontractkit/mcms/internal/core/proposal/mcms"

type BatchChainOperation struct {
	ChainSelector mcms.ChainSelector `json:"chainIdentifier"`
	Batch         []mcms.Operation   `json:"batch"`
}
