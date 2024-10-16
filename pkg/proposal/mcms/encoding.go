package mcms

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	mcm_errors "github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/merkle"
)

func calculateTransactionCounts(transactions []ChainOperation) map[ChainIdentifier]uint64 {
	txCounts := make(map[ChainIdentifier]uint64)
	for _, tx := range transactions {
		txCounts[tx.ChainID]++
	}

	return txCounts
}

func buildEncoders(
	metadatas map[ChainIdentifier]ChainMetadata,
	isSim bool,
) (map[ChainIdentifier]MetadataEncoder, map[ChainIdentifier]OperationEncoder, error) {
	metadataEncoders := make(map[ChainIdentifier]MetadataEncoder)
	opEncoders := make(map[ChainIdentifier]OperationEncoder)
	for chainID, metadata := range metadatas {
		chain, exists := chain_selectors.ChainBySelector(uint64(chainID))
		if !exists {
			return nil, nil, &mcm_errors.ErrInvalidChainID{
				ReceivedChainID: uint64(chainID),
			}
		}

		// Simulated chains always have block.chainid = 1337
		// So for setRoot to execute (not throw WrongChainId) we must
		// override the evmChainID to be 1337.
		if isSim {
			chain.EvmChainID = 1337
		}

		// TODO: this should be a switch statement that generates the
		// chain-specific metadata encoder
		metadataEncoders[chainID] = &EVMMetadataEncoder{
			ChainId: chain.EvmChainID,
		}

		opEncoders[chainID] = &EVMOperationEncoder{
			ChainId:  chain.EvmChainID,
			Multisig: common.HexToAddress(metadata.MCMAddress),
		}
	}

	return metadataEncoders, opEncoders, nil
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
	txCounts map[ChainIdentifier]uint64,
	metadataEncoders map[ChainIdentifier]MetadataEncoder,
	operationsEncoders map[ChainIdentifier]OperationEncoder,
	metadatas map[ChainIdentifier]ChainMetadata,
	ops []ChainOperation,
	overridePreviousRoot bool,
) (*merkle.MerkleTree, error) {
	hashLeaves := make([]common.Hash, 0)
	chainIdx := make(map[ChainIdentifier]uint32, len(metadatas))

	for _, chainID := range chainIdentifiers {
		encodedRootMetadata, err := metadataEncoders[chainID].Hash(metadatas[chainID], txCounts[chainID], overridePreviousRoot)
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedRootMetadata)
	}

	for _, op := range ops {
		encodedOp, err := operationsEncoders[op.ChainID].Hash(op, chainIdx[op.ChainID])
		if err != nil {
			return nil, err
		}
		hashLeaves = append(hashLeaves, encodedOp)
		chainIdx[op.ChainID]++
	}

	// sort the hashes and sort the pairs
	sort.Slice(hashLeaves, func(i, j int) bool {
		return hashLeaves[i].String() < hashLeaves[j].String()
	})

	return merkle.NewMerkleTree(hashLeaves), nil
}
