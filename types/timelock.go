package types

type TimelockOperationType string

const (
	Schedule TimelockOperationType = "schedule"
	Cancel   TimelockOperationType = "cancel"
	Bypass   TimelockOperationType = "bypass"
)

type BatchChainOperation struct {
	ChainIdentifier ChainIdentifier `json:"chainIdentifier"`
	Batch           []Operation     `json:"batch"`
}
