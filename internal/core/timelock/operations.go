package timelock

import "github.com/smartcontractkit/mcms/pkg/proposal/mcms"

type BatchChainOperation struct {
	// Chain identifier is used to map this batch to the correct Chain Encoder
	ChainIdentifier mcms.ChainIdentifier `json:"chainIdentifier"`

	// Operations to be executed, cancelled or bypassed
	Batch []mcms.Operation `json:"batch"`

	// Address of the targetted timelock contract
	TimelockAddress string `json:"timelockAddress"`
}
