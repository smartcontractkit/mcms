package stellar

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Encoder = (*Encoder)(nil)

// Encoder implements sdk.Encoder for the Soroban MCMS contract (Stellar), matching
// chainlink-stellar contracts/mcms ABI leaf hashing.
type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

// NewEncoder returns a new Stellar MCMS encoder.
func NewEncoder(chainSelector types.ChainSelector, txCount uint64, overridePreviousRoot bool) *Encoder {
	return &Encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

// HashOperation implements sdk.Encoder.
func (e *Encoder) HashOperation(
	opCount uint32,
	metadata types.ChainMetadata,
	op types.Operation,
) (common.Hash, error) {
	if uint64(opCount) >= uint40MaxExclusive {
		return common.Hash{}, fmt.Errorf("%w: opCount %d", ErrUint40Overflow, opCount)
	}

	chainID, err := ChainNetworkID(e.ChainSelector)
	if err != nil {
		return common.Hash{}, err
	}

	multisig, err := ParseContractID(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("mcmAddress: %w", err)
	}

	to, err := ParseContractID(op.Transaction.To)
	if err != nil {
		return common.Hash{}, fmt.Errorf("transaction.to: %w", err)
	}

	valueWord, err := parseValueWord(op.Transaction.AdditionalFields)
	if err != nil {
		return common.Hash{}, err
	}

	h, err := HashStellarOp(
		domainOpStellar,
		chainID,
		multisig,
		uint64(opCount),
		to,
		valueWord,
		op.Transaction.Data,
	)
	if err != nil {
		return common.Hash{}, err
	}

	return h, nil
}

// HashMetadata implements sdk.Encoder.
func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	if metadata.StartingOpCount >= uint40MaxExclusive {
		return common.Hash{}, fmt.Errorf("%w: startingOpCount %d", ErrUint40Overflow, metadata.StartingOpCount)
	}
	post := metadata.StartingOpCount + e.TxCount
	if post >= uint40MaxExclusive {
		return common.Hash{}, fmt.Errorf("%w: postOpCount (starting+txCount) %d", ErrUint40Overflow, post)
	}

	chainID, err := ChainNetworkID(e.ChainSelector)
	if err != nil {
		return common.Hash{}, err
	}

	multisig, err := ParseContractID(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("mcmAddress: %w", err)
	}

	return HashRootMetadata(
		domainMetaStellar,
		chainID,
		multisig,
		metadata.StartingOpCount,
		post,
		e.OverridePreviousRoot,
	)
}

// parseValueWord reads optional transaction.additionalFields JSON for StellarOp.value (uint256).
// V1 on-chain requires zero; omit additionalFields or use "{}" unless supplying non-zero value as hex.
func parseValueWord(raw json.RawMessage) ([32]byte, error) {
	var zero [32]byte
	if len(raw) == 0 {
		return zero, nil
	}

	var af struct {
		Value *string `json:"value,omitempty"`
	}
	if err := json.Unmarshal(raw, &af); err != nil {
		return zero, fmt.Errorf("unmarshal stellar additionalFields: %w", err)
	}
	if af.Value == nil || *af.Value == "" {
		return zero, nil
	}

	s := *af.Value
	if len(s) >= hexPrefixLen && (s[0:hexPrefixLen] == "0x" || s[0:hexPrefixLen] == "0X") {
		s = s[hexPrefixLen:]
	}
	if len(s) != stellarChainHexCharLen {
		return zero, fmt.Errorf("value must be 32-byte hex (64 chars), got length %d", len(s))
	}
	n := new(big.Int)
	_, ok := n.SetString(s, hexRadix)
	if !ok {
		return zero, fmt.Errorf("invalid value hex")
	}
	if n.Sign() < 0 || n.BitLen() > uint256BitWidth {
		return zero, fmt.Errorf("value out of uint256 range")
	}
	var out [32]byte
	n.FillBytes(out[:])

	return out, nil
}
