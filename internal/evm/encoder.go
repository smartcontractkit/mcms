package evm

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP"))
var MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA"))

type EVMEncoder struct {
	TxCount              uint64
	ChainID              uint64
	OverridePreviousRoot bool
}

func NewEVMEncoder(txCount uint64, chainID uint64, overridePreviousRoot bool) *EVMEncoder {
	return &EVMEncoder{
		TxCount:              txCount,
		ChainID:              chainID,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

func (e *EVMEncoder) HashOperation(opCount uint32, metadata mcms.ChainMetadata, operation mcms.ChainOperation) (common.Hash, error) {
	// Unmarshall the AdditionalFields from the operation
	additionalFields := EVMAdditionalFields{}
	if err := json.Unmarshal(operation.AdditionalFields, &additionalFields); err != nil {
		return common.Hash{}, err
	}

	op := bindings.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(e.ChainID),
		MultiSig: common.HexToAddress(metadata.MCMAddress),
		Nonce:    new(big.Int).SetUint64(metadata.StartingOpCount + uint64(opCount)),
		To:       common.HexToAddress(operation.To),
		Data:     operation.Data,
		Value:    additionalFields.Value,
	}

	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"nonce","type":"uint40"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP, op)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func (e *EVMEncoder) HashMetadata(metadata mcms.ChainMetadata) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, e.ToGethRootMetadata(metadata))
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func (e *EVMEncoder) ToGethOperation(opCount uint32, metadata mcms.ChainMetadata, operation mcms.ChainOperation) (bindings.ManyChainMultiSigOp, error) {
	// Unmarshall the AdditionalFields from the operation
	additionalFields := EVMAdditionalFields{}
	if err := json.Unmarshal(operation.AdditionalFields, &additionalFields); err != nil {
		return bindings.ManyChainMultiSigOp{}, err
	}

	return bindings.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(e.ChainID),
		MultiSig: common.HexToAddress(metadata.MCMAddress),
		Nonce:    new(big.Int).SetUint64(metadata.StartingOpCount + uint64(opCount)),
		To:       common.HexToAddress(operation.To),
		Data:     operation.Data,
		Value:    additionalFields.Value,
	}, nil
}

func (e *EVMEncoder) ToGethRootMetadata(metadata mcms.ChainMetadata) bindings.ManyChainMultiSigRootMetadata {
	return bindings.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(e.ChainID),
		MultiSig:             common.HexToAddress(metadata.MCMAddress),
		PreOpCount:           new(big.Int).SetUint64(metadata.StartingOpCount),
		PostOpCount:          new(big.Int).SetUint64(metadata.StartingOpCount + e.TxCount),
		OverridePreviousRoot: e.OverridePreviousRoot,
	}
}

// func buildRootMetadatas(
// 	chainMetadata map[ChainIdentifier]ChainMetadata,
// 	txCounts map[ChainIdentifier]uint64,
// 	overridePreviousRoot bool,
// 	isSim bool,
// ) (map[ChainIdentifier]bindings.ManyChainMultiSigRootMetadata, error) {
// 	rootMetadatas := make(map[ChainIdentifier]bindings.ManyChainMultiSigRootMetadata)

// 	for chainID, metadata := range chainMetadata {
// 		chain, exists := chain_selectors.ChainBySelector(uint64(chainID))
// 		if !exists {
// 			return nil, &errors.InvalidChainIDError{
// 				ReceivedChainID: uint64(chainID),
// 			}
// 		}

// 		currentTxCount, ok := txCounts[chainID]
// 		if !ok {
// 			return nil, &errors.MissingChainDetailsError{
// 				ChainIdentifier: uint64(chainID),
// 				Parameter:       "transaction count",
// 			}
// 		}

// 		// Simulated chains always have block.chainid = 1337
// 		// So for setRoot to execute (not throw WrongChainId) we must
// 		// override the evmChainID to be 1337.
// 		if isSim {
// 			chain.EvmChainID = 1337
// 		}

// 		preOpCount := new(big.Int).SetUint64(metadata.StartingOpCount)
// 		postOpCount := new(big.Int).SetUint64(metadata.StartingOpCount + currentTxCount)

// 		rootMetadatas[chainID] = bindings.ManyChainMultiSigRootMetadata{
// 			ChainId:              new(big.Int).SetUint64(chain.EvmChainID),
// 			MultiSig:             metadata.MCMAddress,
// 			PreOpCount:           preOpCount,
// 			PostOpCount:          postOpCount,
// 			OverridePreviousRoot: overridePreviousRoot,
// 		}
// 	}

// 	return rootMetadatas, nil
// }

// func buildOperations(
// 	transactions []ChainOperation,
// 	rootMetadatas map[ChainIdentifier]bindings.ManyChainMultiSigRootMetadata,
// 	txCounts map[ChainIdentifier]uint64,
// ) (map[ChainIdentifier][]bindings.ManyChainMultiSigOp, []bindings.ManyChainMultiSigOp) {
// 	ops := make(map[ChainIdentifier][]bindings.ManyChainMultiSigOp)
// 	chainAgnosticOps := make([]bindings.ManyChainMultiSigOp, 0)
// 	chainIdx := make(map[ChainIdentifier]uint32, len(rootMetadatas))

// 	for _, tx := range transactions {
// 		rootMetadata := rootMetadatas[tx.ChainIdentifier]
// 		if _, ok := ops[tx.ChainIdentifier]; !ok {
// 			ops[tx.ChainIdentifier] = make([]bindings.ManyChainMultiSigOp, txCounts[tx.ChainIdentifier])
// 			chainIdx[tx.ChainIdentifier] = 0
// 		}

// 		op := bindings.ManyChainMultiSigOp{
// 			ChainId:  rootMetadata.ChainId,
// 			MultiSig: rootMetadata.MultiSig,
// 			Nonce:    big.NewInt(rootMetadata.PreOpCount.Int64() + int64(chainIdx[tx.ChainIdentifier])),
// 			To:       tx.To,
// 			Data:     tx.Data,
// 			Value:    tx.Value,
// 		}

// 		chainAgnosticOps = append(chainAgnosticOps, op)
// 		ops[tx.ChainIdentifier][chainIdx[tx.ChainIdentifier]] = op
// 		chainIdx[tx.ChainIdentifier]++
// 	}

// 	return ops, chainAgnosticOps
// }

// func buildMerkleTree(
// 	chainIdentifiers []ChainIdentifier,
// 	rootMetadatas map[ChainIdentifier]bindings.ManyChainMultiSigRootMetadata,
// 	ops map[ChainIdentifier][]bindings.ManyChainMultiSigOp,
// ) (*merkle.Tree, error) {
// 	hashLeaves := make([]common.Hash, 0)

// 	for _, chainID := range chainIdentifiers {
// 		encodedRootMetadata, err := metadataEncoder(rootMetadatas[chainID])
// 		if err != nil {
// 			return nil, err
// 		}
// 		hashLeaves = append(hashLeaves, encodedRootMetadata)

// 		for _, op := range ops[chainID] {
// 			encodedOp, err := txEncoder(op)
// 			if err != nil {
// 				return nil, err
// 			}
// 			hashLeaves = append(hashLeaves, encodedOp)
// 		}
// 	}

// 	// sort the hashes and sort the pairs
// 	sort.Slice(hashLeaves, func(i, j int) bool {
// 		return hashLeaves[i].String() < hashLeaves[j].String()
// 	})

// 	return merkle.NewMerkleTree(hashLeaves), nil
// }
