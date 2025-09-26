package ton

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// TODO: import from github.com/smartcontractkit/chainlink-ton/bindings/mcms/mcms
// TODO: update "MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR...", add TON prefixes
var (
	mcmDomainSeparatorOp       = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_TON"))
	mcmDomainSeparatorMetadata = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_TON"))
)

var _ sdk.Encoder = &encoder{}

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
	return common.Hash{}, fmt.Errorf("not implemented")
}
