package mcms

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/pkg/gethwrappers"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

func calculateTransactionCounts(transactions []ChainOperation) map[ChainIdentifier]uint64 {
	txCounts := make(map[ChainIdentifier]uint64)
	for _, tx := range transactions {
		txCounts[tx.ChainID]++
	}

	return txCounts
}

func buildOperations(
	transactions []ChainOperation,
	rootMetadatas ChainMetadatas,
	txCounts map[ChainIdentifier]uint64,
	overridePreviousRoot bool,
	sim bool,
) (map[ChainIdentifier][]gethwrappers.ManyChainMultiSigOp, []gethwrappers.ManyChainMultiSigOp) {
	ops := make(map[ChainIdentifier][]gethwrappers.ManyChainMultiSigOp)
	chainAgnosticOps := make([]gethwrappers.ManyChainMultiSigOp, 0)
	chainIdx := make(map[ChainIdentifier]uint32, len(rootMetadatas))

	for _, tx := range transactions {
		rootMetadata := rootMetadatas[tx.ChainID]
		if _, ok := ops[tx.ChainID]; !ok {
			ops[tx.ChainID] = make([]gethwrappers.ManyChainMultiSigOp, txCounts[tx.ChainID])
			chainIdx[tx.ChainID] = 0
		}

		op := gethwrappers.ManyChainMultiSigOp{
			ChainId:  rootMetadata.ChainId,
			MultiSig: rootMetadata.MultiSig,
			Nonce:    big.NewInt(rootMetadata.PreOpCount.Int64() + int64(chainIdx[tx.ChainID])),
			To:       tx.To,
			Data:     tx.Data,
			Value:    tx.Value,
		}

		chainAgnosticOps = append(chainAgnosticOps, op)
		ops[tx.ChainID][chainIdx[tx.ChainID]] = op
		chainIdx[tx.ChainID]++
	}

	return ops, chainAgnosticOps
}

func sortedChainIdentifiers(chainMetadata map[ChainIdentifier]ChainMetadata) []ChainIdentifier {
	chainIdentifiers := make([]ChainIdentifier, 0, len(chainMetadata))
	for chainID := range chainMetadata {
		chainIdentifiers = append(chainIdentifiers, chainID)
	}
	sort.Slice(chainIdentifiers, func(i, j int) bool { return chainIdentifiers[i] < chainIdentifiers[j] })

	return chainIdentifiers
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
