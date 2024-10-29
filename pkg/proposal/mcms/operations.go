package mcms

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
