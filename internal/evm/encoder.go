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
	rootMetadata := bindings.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(e.ChainID),
		MultiSig:             common.HexToAddress(metadata.MCMAddress),
		PreOpCount:           new(big.Int).SetUint64(metadata.StartingOpCount),
		PostOpCount:          new(big.Int).SetUint64(metadata.StartingOpCount + e.TxCount),
		OverridePreviousRoot: e.OverridePreviousRoot,
	}

	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, rootMetadata)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func metadataEncoder(rootMetadata bindings.ManyChainMultiSigRootMetadata) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA, rootMetadata)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}

func txEncoder(op bindings.ManyChainMultiSigOp) (common.Hash, error) {
	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"nonce","type":"uint40"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}]}]`
	encoded, err := ABIEncode(abi, MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP, op)
	if err != nil {
		return common.Hash{}, err
	}

	return crypto.Keccak256Hash(encoded), nil
}
