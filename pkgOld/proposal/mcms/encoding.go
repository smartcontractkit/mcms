package mcms

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

func buildRootMetadatas(
	chainMetadata map[ChainIdentifier]ChainMetadata,
	txCounts map[ChainIdentifier]uint64,
	overridePreviousRoot bool,
	isSim bool,
) (map[ChainIdentifier]gethwrappers.ManyChainMultiSigRootMetadata, error) {
	rootMetadatas := make(map[ChainIdentifier]gethwrappers.ManyChainMultiSigRootMetadata)

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
	rootMetadatas map[ChainIdentifier]gethwrappers.ManyChainMultiSigRootMetadata,
	txCounts map[ChainIdentifier]uint64,
) (map[ChainIdentifier][]gethwrappers.ManyChainMultiSigOp, []gethwrappers.ManyChainMultiSigOp) {
	ops := make(map[ChainIdentifier][]gethwrappers.ManyChainMultiSigOp)
	chainAgnosticOps := make([]gethwrappers.ManyChainMultiSigOp, 0)
	chainIdx := make(map[ChainIdentifier]uint32, len(rootMetadatas))

	for _, tx := range transactions {
		rootMetadata := rootMetadatas[tx.ChainIdentifier]
		if _, ok := ops[tx.ChainIdentifier]; !ok {
			ops[tx.ChainIdentifier] = make([]gethwrappers.ManyChainMultiSigOp, txCounts[tx.ChainIdentifier])
			chainIdx[tx.ChainIdentifier] = 0
		}

		op := gethwrappers.ManyChainMultiSigOp{
			ChainId:  rootMetadata.ChainId,
			MultiSig: rootMetadata.MultiSig,
			Nonce:    big.NewInt(rootMetadata.PreOpCount.Int64() + int64(chainIdx[tx.ChainIdentifier])),
			To:       tx.To,
			Data:     tx.Data,
			Value:    tx.Value,
		}

		chainAgnosticOps = append(chainAgnosticOps, op)
		ops[tx.ChainIdentifier][chainIdx[tx.ChainIdentifier]] = op
		chainIdx[tx.ChainIdentifier]++
	}

	return ops, chainAgnosticOps
}

func buildMerkleTree(
	chainIdentifiers []ChainIdentifier,
	rootMetadatas map[ChainIdentifier]gethwrappers.ManyChainMultiSigRootMetadata,
	ops map[ChainIdentifier][]gethwrappers.ManyChainMultiSigOp,
) (*merkle.MerkleTree, error) {
	hashLeaves := make([]common.Hash, 0)

	for _, chainID := range chainIdentifiers {
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
