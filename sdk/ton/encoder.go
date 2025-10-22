package ton

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/ocr"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: mv and import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
// TODO: update "MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR...", add TON prefixes
var (
	mcmDomainSeparatorOp       = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_TON"))
	mcmDomainSeparatorMetadata = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_TON"))
)

var _ sdk.Encoder = &encoder{}

// Implementations of various encoding interfaces for TON MCMS
var _ RootMetadataEncoder[mcms.RootMetadata] = &encoder{}
var _ OperationEncoder[mcms.Op] = &encoder{}
var _ ProofEncoder[mcms.Proof] = &encoder{}
var _ SignaturesEncoder[ocr.SignatureEd25519] = &encoder{}

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

type encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

// Encoder encoding MCMS operations and metadata into hashes.
func NewEncoder(
	chainSelector types.ChainSelector,
	txCount uint64,
	overridePreviousRoot bool,
) sdk.Encoder {
	return &encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

func (e *encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	chainID, err := chain_selectors.TonChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get chain ID from selector: %w", err)
	}

	// Map to Ton Address type (mcms.address)
	mcmsAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	// Map to Ton Address type (op.to)
	toAddr, err := address.ParseAddr(op.Transaction.To)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid op.Transaction.To address: %w", err)
	}

	// Encode operation according to TON specs
	// TODO: unpack configured value
	var value tlb.Coins
	// TODO: unpack op.Transaction.Data,
	var data *cell.Cell

	opCell, err := tlb.ToCell(mcms.Op{
		ChainID:  (&big.Int{}).SetInt64(int64(chainID)),
		MultiSig: mcmsAddr,
		Nonce:    uint64(opCount),
		To:       toAddr,
		Value:    value,
		Data:     data,
	})
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

func (e *encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
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

func (e *encoder) ToOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (mcms.Op, error) {
	chainID, err := chain_selectors.TonChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return mcms.Op{}, fmt.Errorf("failed to get chain ID from selector: %w", err)
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

func (e *encoder) ToRootMetadata(metadata types.ChainMetadata) (mcms.RootMetadata, error) {
	chainID, err := chain_selectors.TonChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return mcms.RootMetadata{}, fmt.Errorf("failed to get chain ID from selector: %w", err)
	}

	// Map to Ton Address type (mcms.address)
	mcmsAddr, err := address.ParseAddr(metadata.MCMAddress)
	if err != nil {
		return mcms.RootMetadata{}, fmt.Errorf("invalid mcms address: %w", err)
	}

	return mcms.RootMetadata{
		ChainID:              (&big.Int{}).SetInt64(int64(chainID)),
		MultiSig:             *mcmsAddr,
		PreOpCount:           metadata.StartingOpCount,
		PostOpCount:          metadata.StartingOpCount + e.TxCount,
		OverridePreviousRoot: e.OverridePreviousRoot,
	}, nil
}

func (e *encoder) ToProof(p []common.Hash) ([]mcms.Proof, error) {
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

func (e *encoder) ToSignatures(ss []types.Signature, hash common.Hash) ([]ocr.SignatureEd25519, error) {
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
			// TODO: use [32]byte arrays
			R:      []byte(s.R.Bytes()),
			S:      []byte(s.S.Bytes()),
			Signer: pubKeyBytes,
		})
	}

	return bindSignatures, nil
}
