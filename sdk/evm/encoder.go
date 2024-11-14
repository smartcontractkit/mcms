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
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
	IsSim                bool
}

// NewEncoder returns a new Encoder.
func NewEncoder(
	csel types.ChainSelector, txCount uint64, overridePreviousRoot bool, isSim bool,
) *Encoder {
	return &Encoder{
		ChainSelector:        csel,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
		IsSim:                isSim,
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
	bindMeta, err := e.ToGethRootMetadata(metadata)
	if err != nil {
		return common.Hash{}, err
	}

	abi := `[{"type":"bytes32"},{"type":"tuple","components":[{"name":"chainId","type":"uint256"},{"name":"multiSig","type":"address"},{"name":"preOpCount","type":"uint40"},{"name":"postOpCount","type":"uint40"},{"name":"overridePreviousRoot","type":"bool"}]}]`
	encoded, err := abiUtils.ABIEncode(abi, mcmDomainSeparatorMetadata, bindMeta)
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
	evmChainID, err := getEVMChainID(e.ChainSelector, e.IsSim)
	if err != nil {
		return bindings.ManyChainMultiSigOp{}, err
	}

	// Unmarshal the AdditionalFields from the operation
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return bindings.ManyChainMultiSigOp{}, err
	}

	return bindings.ManyChainMultiSigOp{
		ChainId:  new(big.Int).SetUint64(evmChainID),
		MultiSig: common.HexToAddress(metadata.MCMAddress),
		Nonce:    new(big.Int).SetUint64(metadata.StartingOpCount + uint64(opCount)),
		To:       common.HexToAddress(op.Transaction.To),
		Data:     op.Transaction.Data,
		Value:    additionalFields.Value,
	}, nil
}

// ToGethRootMetadata converts the MCMS ChainMetadata into the format expected by the EVM
// ManyChainMultiSig contract.
func (e *Encoder) ToGethRootMetadata(metadata types.ChainMetadata) (bindings.ManyChainMultiSigRootMetadata, error) {
	evmChainID, err := getEVMChainID(e.ChainSelector, e.IsSim)
	if err != nil {
		return bindings.ManyChainMultiSigRootMetadata{}, err
	}

	return bindings.ManyChainMultiSigRootMetadata{
		ChainId:              new(big.Int).SetUint64(evmChainID),
		MultiSig:             common.HexToAddress(metadata.MCMAddress),
		PreOpCount:           new(big.Int).SetUint64(metadata.StartingOpCount),
		PostOpCount:          new(big.Int).SetUint64(metadata.StartingOpCount + e.TxCount),
		OverridePreviousRoot: e.OverridePreviousRoot,
	}, nil
}
