package mcms

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

type RootMetadata[R any] interface {
	Verbose(txCount uint64) R
	Hash(txCount uint64) (common.Hash, error)
}

type ChainMetadatas map[ChainIdentifier]RootMetadata[any]

func (c *ChainMetadatas) UnmarshalJSON(data []byte) error {
	// Step 1: Define a temporary struct for the fields we want to unmarshal first
	tempMap := map[ChainIdentifier]json.RawMessage{}

	// Unmarshal only the ChainID
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return err
	}

	// TODO: Step 2: Unmarshal the RootMetadata field based on ChainID

	return nil
}

type EVMChainMetadata struct {
	StartingOpCount uint64         `json:"startingOpCount"`
	MCMAddress      common.Address `json:"mcmAddress"`
}

func (e *EVMChainMetadata) Verbose(txCount uint64) gethwrappers.ManyChainMultiSigRootMetadata {
	// TODO: Implement
	return gethwrappers.ManyChainMultiSigRootMetadata{}
}

func (e *EVMChainMetadata) Hash(txCount uint64) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, e.Verbose())
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

type ExampleChainMetadata struct {
	StartingOpCount uint64 `json:"startingOpCount"`
	MCMAddress      string `json:"mcmAddress"`
}

func (e *ExampleChainMetadata) Verbose(txCount uint64) struct{} {
	return struct{}{}
}

func (e *ExampleChainMetadata) Hash(txCount uint64) (common.Hash, error) {
	return common.Hash{}, nil
}
