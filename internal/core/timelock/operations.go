package timelock

import "github.com/smartcontractkit/mcms/pkg/proposal/mcms"

type BatchChainOperation struct {
	// TODO: Why is chainIDentifier needed?
	ChainIdentifier mcms.ChainIdentifier `json:"chainIdentifier"`

	// Operations to be executed, cancelled or bypassed
	Batch []mcms.Operation `json:"batch"`

	// Address of the targetted timelock contract
	TimelockAddress string `json:"timelockAddress"`
}
