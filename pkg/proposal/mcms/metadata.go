package mcms

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
)

var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA"))

type RootMetadata[R any] interface {
	Verbose(chainID uint64, txCount uint64, overridePreviousRoot bool) R
	Hash(chainID uint64, txCount uint64, overridePreviousRoot bool) (common.Hash, error)
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

// TODO: this might not be necessary
func (c *ChainMetadatas) Verbose(
	txCounts map[ChainIdentifier]uint64,
	overridePreviousRoot bool,
	isSim bool,
) (map[ChainIdentifier]any, error) {
	rootMetadatas := make(map[ChainIdentifier]any)

	for chainID, metadata := range *c {
		chain, exists := chain_selectors.ChainBySelector(uint64(chainID))
		if !exists {
			return nil, &errors.ErrInvalidChainID{
				ReceivedChainID: uint64(chainID),
			}
		}

		currentTxCount, ok := txCounts[chainID]
		if !ok {
			return nil, &errors.ErrMissingChainDetails{
				ChainIdentifier: uint64(chainID),
				Parameter:       "transaction count",
			}
		}

		// Simulated chains always have block.chainid = 1337
		// So for setRoot to execute (not throw WrongChainId) we must
		// override the evmChainID to be 1337.
		if isSim {
			chain.EvmChainID = 1337
		}

		rootMetadatas[chainID] = metadata.Verbose(chain.EvmChainID, currentTxCount, overridePreviousRoot)
	}

	return rootMetadatas, nil
}

type EVMChainMetadata struct {
	StartingOpCount uint64         `json:"startingOpCount"`
	MCMAddress      common.Address `json:"mcmAddress"`
}

func (e *EVMChainMetadata) Verbose(chainID uint64, txCount uint64, overridePreviousRoot bool) gethwrappers.ManyChainMultiSigRootMetadata {
	// TODO: Implement
	return gethwrappers.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(chainID),
		MultiSig:             e.MCMAddress,
		PreOpCount:           new(big.Int).SetUint64(e.StartingOpCount),
		PostOpCount:          new(big.Int).SetUint64(e.StartingOpCount + txCount),
		OverridePreviousRoot: overridePreviousRoot,
	}
}

func (e *EVMChainMetadata) Hash(chainID uint64, txCount uint64, overridePreviousRoot bool) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, e.Verbose(chainID, txCount, overridePreviousRoot))
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

func (e *ExampleChainMetadata) Hash(chainID uint64, txCount uint64, overridePreviousRoot bool) (common.Hash, error) {
	return common.Hash{}, nil
}
