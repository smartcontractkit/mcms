package mcms

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

type ChainIdentifier uint64

type Operation[R any] interface {
	Verbose() R
	Hash() (common.Hash, error)
}

type OperationMetadata struct {
	ContractType string   `json:"contractType"`
	Tags         []string `json:"tags"`
}

type ChainOperation struct {
	ChainID ChainIdentifier `json:"chainIdentifier"`
	Operation[any]
	OperationMetadata
}

func (co *ChainOperation) UnmarshalJSON(data []byte) error {
	// Step 1: Define a temporary struct for the fields we want to unmarshal first
	var temp struct {
		ChainID           ChainIdentifier   `json:"chainIdentifier"`
		OperationMetadata OperationMetadata `json:"operationMetadata"`
	}

	// Unmarshal only the ChainID and OperationMetadata
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Step 2: Set ChainID and OperationMetadata on the actual ChainOperation struct
	co.ChainID = temp.ChainID
	co.OperationMetadata = temp.OperationMetadata

	// Step 3: Unmarshal the Operation field based on ChainID
	// TODO: Implement

	return nil
}

type EVMOperation struct {
	To    common.Address `json:"to"`
	Data  []byte         `json:"data"`
	Value *big.Int       `json:"value"`
}

func (e *EVMOperation) Verbose() gethwrappers.ManyChainMultiSigOp {
	// TODO: Implement
	return gethwrappers.ManyChainMultiSigOp{}
}

func (e *EVMOperation) Hash() (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"nonce","type":"uint40"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP, e.Verbose())
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

type ExampleChainOperation struct {
	To    string `json:"to"`
	Data  []byte `json:"data"`
	Value uint64 `json:"value"`
}

func (e *ExampleChainOperation) Verbose() struct{} {
	return struct{}{}
}

func (e *ExampleChainOperation) Hash() (common.Hash, error) {
	return common.Hash{}, nil
}
