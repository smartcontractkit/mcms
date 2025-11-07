package ton

import (
	"encoding/json"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/ocr"

	"github.com/smartcontractkit/mcms/sdk"
	sdkerrors "github.com/smartcontractkit/mcms/sdk/errors"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
// TODO: a different hash fn is used in TON sha256
var (
	mcmDomainSeparatorOp       = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_TON"))
	mcmDomainSeparatorMetadata = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_TON"))
)

var _ sdk.Encoder = &Encoder{}

// Implementations of various encoding interfaces for TON MCMS
var _ RootMetadataEncoder[mcms.RootMetadata] = &Encoder{}
var _ OperationEncoder[mcms.Op] = &Encoder{}
var _ ProofEncoder[mcms.Proof] = &Encoder{}
var _ SignaturesEncoder[ocr.SignatureEd25519] = &Encoder{}

// TODO: bubble up to sdk, use in evm as well
// Defines encoding from sdk types.ChainMetadata to chain type RootMetadata T
type RootMetadataEncoder[T any] interface {
	ToRootMetadata(metadata types.ChainMetadata) (T, error)
}

// TODO: bubble up to sdk, use in evm as well
// Defines encoding from sdk types.ChainMetadata + types.Operation to chain type Operation T
type OperationEncoder[T any] interface {
	ToOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (T, error)
}

// TODO: bubble up to sdk, use in evm as well
// Defines encoding from sdk []common.Hash to chain type Proof []T
type ProofEncoder[T any] interface {
	ToProof(p []common.Hash) ([]T, error)
}

// TODO: bubble up to sdk, use in evm as well
// Defines encoding from sdk []types.Signature to chain type Signature []T
type SignaturesEncoder[T any] interface {
	ToSignatures(s []types.Signature, hash common.Hash) ([]T, error)
}

// Encoder encoding MCMS operations and metadata into hashes.
type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

func NewEncoder(chainSelector types.ChainSelector, txCount uint64, overridePreviousRoot bool) sdk.Encoder {
	return &Encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	opBind, err := e.ToOperation(opCount, metadata, op)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to convert operation: %w", err)
	}

	opCell, err := tlb.ToCell(opBind)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode op: %w", err)
	}

	// Hash operation according to TON specs
	// @dev we use the standard sha256 (cell) hash function to hash the leaf.
	b := cell.BeginCell()
	if err := b.StoreBigUInt(new(big.Int).SetBytes(mcmDomainSeparatorOp), 256); err != nil {
		return common.Hash{}, fmt.Errorf("failed to store domain separator: %w", err)
	}
	if err := b.StoreRef(opCell); err != nil {
		return common.Hash{}, fmt.Errorf("failed to store op cell ref: %w", err)
	}

	var hash common.Hash
	copy(hash[:], b.EndCell().Hash()[:32])
	return hash, nil
}

func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	rm, err := e.ToRootMetadata(metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to convert to root metadata: %w", err)
	}

	// Encode metadata according to TON specs
	metaCell, err := tlb.ToCell(rm)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to encode op: %w", err)
	}

	// Hash metadata according to TON specs
	// @dev we use the standard sha256 (cell) hash function to hash the leaf.
	b := cell.BeginCell()
	if err := b.StoreBigUInt(new(big.Int).SetBytes(mcmDomainSeparatorMetadata), 256); err != nil {
		return common.Hash{}, fmt.Errorf("failed to store domain separator: %w", err)
	}
	if err := b.StoreRef(metaCell); err != nil {
		return common.Hash{}, fmt.Errorf("failed to store meta cell ref: %w", err)
	}

	var hash common.Hash
	copy(hash[:], b.EndCell().Hash()[:32])
	return hash, nil
}

func (e *Encoder) ToOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (mcms.Op, error) {
	chainID, err := chain_selectors.TonChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return mcms.Op{}, &sdkerrors.InvalidChainIDError{ReceivedChainID: e.ChainSelector}
	}

	// Unmarshal the AdditionalFields from the operation
	var additionalFields AdditionalFields
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return mcms.Op{}, err
	}

	// Map to Ton Address types
	mcmsAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return mcms.Op{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	toAddr, err := address.ParseAddr(op.Transaction.To)
	if err != nil {
		return mcms.Op{}, fmt.Errorf("invalid to address: %w", err)
	}

	datac, err := cell.FromBOC(op.Transaction.Data)
	if err != nil {
		return mcms.Op{}, fmt.Errorf("invalid cell BOC data: %w", err)
	}

	return mcms.Op{
		ChainID:  (&big.Int{}).SetInt64(int64(chainID)),
		MultiSig: mcmsAddr,
		Nonce:    uint64(opCount),
		To:       toAddr,
		Data:     datac,
		Value:    tlb.FromNanoTON(additionalFields.Value),
	}, nil
}

func (e *Encoder) ToRootMetadata(metadata types.ChainMetadata) (mcms.RootMetadata, error) {
	chainID, err := chain_selectors.TonChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return mcms.RootMetadata{}, &sdkerrors.InvalidChainIDError{ReceivedChainID: e.ChainSelector}
	}

	// Map to Ton Address type (mcms.address)
	mcmsAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return mcms.RootMetadata{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	return mcms.RootMetadata{
		ChainID:              (&big.Int{}).SetInt64(int64(chainID)),
		MultiSig:             mcmsAddr,
		PreOpCount:           metadata.StartingOpCount,
		PostOpCount:          metadata.StartingOpCount + e.TxCount,
		OverridePreviousRoot: e.OverridePreviousRoot,
	}, nil
}

func (e *Encoder) ToProof(p []common.Hash) ([]mcms.Proof, error) {
	proofs := make([]mcms.Proof, 0, len(p))
	for _, hash := range p {
		proofs = append(proofs, mcms.Proof{Value: hash.Big()})
	}
	return proofs, nil
}

const (
	SignatureVOffset    = 27
	SignatureVThreshold = 2
)

func (e *Encoder) ToSignatures(ss []types.Signature, hash common.Hash) ([]ocr.SignatureEd25519, error) {
	bindSignatures := make([]ocr.SignatureEd25519, 0, len(ss))
	for _, s := range ss {
		if s.V < SignatureVThreshold {
			s.V += SignatureVOffset
		}

		// Notice: to verify the signature on TON we need to recover/publish the public key
		pubKey, err := s.RecoverPublicKey(hash)
		if err != nil {
			return []ocr.SignatureEd25519{}, fmt.Errorf("failed to recover public key: %w", err)
		}

		pubKeyBytes := crypto.FromECDSAPub(pubKey)
		bindSignatures = append(bindSignatures, ocr.SignatureEd25519{
			Data: slices.Concat(s.R.Bytes(), s.S.Bytes(), pubKeyBytes),
		})
	}

	return bindSignatures, nil
}
