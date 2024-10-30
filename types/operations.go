package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ChainIdentifier uint64

type Operation struct {
	To               common.Address `json:"to"`
	Data             []byte         `json:"data"`
	Value            *big.Int       `json:"value"` // TODO: remove after interface refactor, this is evm specific
	AdditionalFields json.RawMessage
	ContractType     string   `json:"contractType"`
	Tags             []string `json:"tags"`
}

type ChainOperation struct {
	ChainIdentifier `json:"chainIdentifier"`
	Operation
}

type BatchChainOperation struct {
	// Chain identifier is used to map this batch to the correct Chain Encoder
	ChainIdentifier ChainIdentifier `json:"chainIdentifier"`

	// Operations to be executed, cancelled or bypassed
	Batch []Operation `json:"batch"`

	// Address of the targeted timelock contract
	TimelockAddress string `json:"timelockAddress"`
}
