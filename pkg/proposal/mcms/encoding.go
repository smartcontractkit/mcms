package mcms

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP"))
var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA"))

func calculateTransactionCounts(transactions []ChainOperation) map[ChainSelector]uint64 {
	txCounts := make(map[ChainSelector]uint64)
	for _, tx := range transactions {
		txCounts[tx.ChainSelector]++
	}

	return txCounts
}

func buildRootMetadatas(
	chainMetadata map[ChainSelector]ChainMetadata,
	txCounts map[ChainSelector]uint64,
	overridePreviousRoot bool,
	isSim bool,
) (map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata, error) {
	rootMetadatas := make(map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata)

	for chainID, metadata := range chainMetadata {
		chain, exists := chain_selectors.ChainBySelector(uint64(chainID))
		if !exists {
			return nil, &errors.InvalidChainIDError{
				ReceivedChainID: uint64(chainID),
			}
		}

		currentTxCount, ok := txCounts[chainID]
		if !ok {
			return nil, &errors.MissingChainDetailsError{
				ChainSelector: uint64(chainID),
				Parameter:     "transaction count",
			}
		}

		// Simulated chains always have block.chainid = 1337
		// So for setRoot to execute (not throw WrongChainId) we must
		// override the evmChainID to be 1337.
		if isSim {
			chain.EvmChainID = 1337
		}

		preOpCount := new(big.Int).SetUint64(metadata.StartingOpCount)
		postOpCount := new(big.Int).SetUint64(metadata.StartingOpCount + currentTxCount)

		rootMetadatas[chainID] = gethwrappers.ManyChainMultiSigRootMetadata{
			ChainId:              new(big.Int).SetUint64(chain.EvmChainID),
			MultiSig:             metadata.MCMAddress,
			PreOpCount:           preOpCount,
			PostOpCount:          postOpCount,
			OverridePreviousRoot: overridePreviousRoot,
		}
	}

	return rootMetadatas, nil
}

func buildOperations(
	transactions []ChainOperation,
	rootMetadatas map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata,
	txCounts map[ChainSelector]uint64,
) (map[ChainSelector][]gethwrappers.ManyChainMultiSigOp, []gethwrappers.ManyChainMultiSigOp) {
	ops := make(map[ChainSelector][]gethwrappers.ManyChainMultiSigOp)
	chainAgnosticOps := make([]gethwrappers.ManyChainMultiSigOp, 0)
	chainIdx := make(map[ChainSelector]uint32, len(rootMetadatas))

	for _, tx := range transactions {
		rootMetadata := rootMetadatas[tx.ChainSelector]
		if _, ok := ops[tx.ChainSelector]; !ok {
			ops[tx.ChainSelector] = make([]gethwrappers.ManyChainMultiSigOp, txCounts[tx.ChainSelector])
			chainIdx[tx.ChainSelector] = 0
		}

		op := gethwrappers.ManyChainMultiSigOp{
			ChainId:  rootMetadata.ChainId,
			MultiSig: rootMetadata.MultiSig,
			Nonce:    big.NewInt(rootMetadata.PreOpCount.Int64() + int64(chainIdx[tx.ChainSelector])),
			To:       tx.To,
			Data:     tx.Data,
			Value:    tx.Value,
		}

		chainAgnosticOps = append(chainAgnosticOps, op)
		ops[tx.ChainSelector][chainIdx[tx.ChainSelector]] = op
		chainIdx[tx.ChainSelector]++
	}

	return ops, chainAgnosticOps
}

func sortedChainSelectors(chainMetadata map[ChainSelector]ChainMetadata) []ChainSelector {
	selectors := make([]ChainSelector, 0, len(chainMetadata))
	for chainID := range chainMetadata {
		selectors = append(selectors, chainID)
	}
	sort.Slice(selectors, func(i, j int) bool { return selectors[i] < selectors[j] })

	return selectors
}

func buildMerkleTree(
	selectors []ChainSelector,
	rootMetadatas map[ChainSelector]gethwrappers.ManyChainMultiSigRootMetadata,
	ops map[ChainSelector][]gethwrappers.ManyChainMultiSigOp,
) (*merkle.MerkleTree, error) {
	hashLeaves := make([]common.Hash, 0)

	for _, chainID := range selectors {
		encodedRootMetadata, err := metadataEncoder(rootMetadatas[chainID])
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedRootMetadata)

		for _, op := range ops[chainID] {
			encodedOp, err := txEncoder(op)
			if err != nil {
				return nil, err
			}
			hashLeaves = append(hashLeaves, encodedOp)
		}
	}

	// sort the hashes and sort the pairs
	sort.Slice(hashLeaves, func(i, j int) bool {
		return hashLeaves[i].String() < hashLeaves[j].String()
	})

	return merkle.NewMerkleTree(hashLeaves), nil
}

func metadataEncoder(rootMetadata gethwrappers.ManyChainMultiSigRootMetadata) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, rootMetadata)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func txEncoder(op gethwrappers.ManyChainMultiSigOp) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"nonce","type":"uint40"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP, op)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}
