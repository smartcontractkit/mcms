package timelock

import "github.com/smartcontractkit/mcms/types"

type BatchChainOperation struct {
	// Chain identifier is used to map this batch to the correct Chain Encoder
	ChainIdentifier types.ChainIdentifier `json:"chainIdentifier"`

	// Operations to be executed, cancelled or bypassed
	Batch []types.Operation `json:"batch"`

	// Address of the targetted timelock contract
	TimelockAddress string `json:"timelockAddress"`
}
