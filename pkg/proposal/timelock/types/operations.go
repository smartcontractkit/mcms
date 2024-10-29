package types

import (
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
)

type BatchChainOperation struct {
	ChainIdentifier types.ChainIdentifier `json:"chainIdentifier"`
	Batch           []types.Operation     `json:"batch"`
}
