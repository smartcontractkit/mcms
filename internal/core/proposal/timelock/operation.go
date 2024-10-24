package timelock

import "github.com/smartcontractkit/mcms/internal/core/proposal/mcms"

type BatchChainOperation struct {
	ChainIdentifier mcms.ChainSelector `json:"chainSelector"`
	Transactions    []mcms.Operation   `json:"transactions"`
}
