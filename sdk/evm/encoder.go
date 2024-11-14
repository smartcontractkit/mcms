package evm

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	abiUtils "github.com/smartcontractkit/mcms/internal/utils/abi"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var (
	// mcmDomainSeparatorOp is used for domain separation of the different op values stored in the
	// Merkle tree. This is defined in the ManyChainMultiSig contract.
	//
	// https://github.com/smartcontractkit/ccip-owner-contracts/blob/main/src/ManyChainMultiSig.sol#L11
	mcmDomainSeparatorOp = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP"))

	// mcmDomainSeparatorMetadata is used for domain separation of the different metadata values
	// stored in the Merkle tree. This is defined in the ManyChainMultiSig contract.
	//
	// https://github.com/smartcontractkit/ccip-owner-contracts/blob/main/src/ManyChainMultiSig.sol#L17
	mcmDomainSeparatorMetadata = crypto.Keccak256Hash([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA"))
)

var _ sdk.Encoder = (*Encoder)(nil)

// Encoder is a struct that encodes ChainOperations and ChainMetadata into the format expected
// by the EVM ManyChainMultiSig contract.
type Encoder struct {
	TxCount              uint64
	ChainID              uint64
	OverridePreviousRoot bool
}

// NewEncoder returns a new Encoder.
func NewEncoder(txCount uint64, chainID uint64, overridePreviousRoot bool) *Encoder {
	return &Encoder{
		TxCount:              txCount,
		ChainID:              chainID,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

// HashOperation converts the MCMS Operation into the format expected by the EVM
// ManyChainMultiSig contract, and hashes it.
func (e *Encoder) HashOperation(
	opCount uint32,
	metadata types.ChainMetadata,
	op types.Operation,
) (common.Hash, error) {
	bindOp, err := e.ToGethOperation(opCount, metadata, op)
	if err != nil {
		return common.Hash{}, err
	}

	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"nonce","type":"uint40"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}]}]`
	encoded, err := abiUtils.ABIEncode(abi, mcmDomainSeparatorOp, bindOp)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

// HashMetadata converts the MCMS ChainMetadata into the format expected by the EVM
// ManyChainMultiSig contract, and hashes it.
func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := abiUtils.ABIEncode(abi, mcmDomainSeparatorMetadata, e.ToGethRootMetadata(metadata))
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

// ToGethOperation converts the MCMS ChainOperation into the format expected by the EVM
// ManyChainMultiSig contract.
func (e *Encoder) ToGethOperation(
	opCount uint32,
	metadata types.ChainMetadata,
	op types.Operation,
) (bindings.ManyChainMultiSigOp, error) {
	// Unmarshal the AdditionalFields from the operation
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return bindings.ManyChainMultiSigOp{}, err
	}

	return bindings.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(e.ChainID),
		MultiSig: common.HexToAddress(metadata.MCMAddress),
		Nonce:    new(big.Int).SetUint64(metadata.StartingOpCount + uint64(opCount)),
		To:       common.HexToAddress(op.Transaction.To),
		Data:     op.Transaction.Data,
		Value:    additionalFields.Value,
	}, nil
}

// ToGethRootMetadata converts the MCMS ChainMetadata into the format expected by the EVM
// ManyChainMultiSig contract.
func (e *Encoder) ToGethRootMetadata(metadata types.ChainMetadata) bindings.ManyChainMultiSigRootMetadata {
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
