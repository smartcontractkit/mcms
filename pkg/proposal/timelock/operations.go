package timelock

import "github.com/smartcontractkit/mcms/pkg/proposal/mcms"

type BatchChainOperation struct {
	ChainSelector mcms.ChainSelector `json:"chainSelector"`
	Batch         []mcms.Operation   `json:"batch"`
}
