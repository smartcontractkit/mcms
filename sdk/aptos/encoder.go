package aptos

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var (
	mcmDomainSeparatorOp       = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_OP_APTOS"))
	mcmDomainSeparatorMetadata = crypto.Keccak256([]byte("MANY_CHAIN_MULTI_SIG_DOMAIN_SEPARATOR_METADATA_APTOS"))
)

var _ sdk.Encoder = &Encoder{}

type Encoder struct {
	ChainSelector        types.ChainSelector
	TxCount              uint64
	OverridePreviousRoot bool
}

func NewEncoder(
	chainSelector types.ChainSelector,
	txCount uint64,
	overridePreviousRoot bool,
) *Encoder {
	return &Encoder{
		ChainSelector:        chainSelector,
		TxCount:              txCount,
		OverridePreviousRoot: overridePreviousRoot,
	}
}

//nolint:mnd // Padding to 32 and 64 bytes respectively
func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, err
	}
	mcmsAddress, err := hexToAddress(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to parse MCMS address %q: %w", metadata.MCMAddress, err)
	}
	toAddress, err := hexToAddress(op.Transaction.To)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to parse To address %q: %w", op.Transaction.To, err)
	}
	additionalFields := AdditionalFields{}
	if err := json.Unmarshal(op.Transaction.AdditionalFields, &additionalFields); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}

	var preImage []byte
	preImage = append(preImage, mcmDomainSeparatorOp...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, chainID), 32)...)
	preImage = append(preImage, mcmsAddress[:]...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint32(nil, opCount), 32)...)
	preImage = append(preImage, toAddress[:]...)
	preImage = append(preImage, common.LeftPadBytes([]byte(additionalFields.ModuleName), 64)...)
	preImage = append(preImage, common.LeftPadBytes([]byte(additionalFields.Function), 64)...)
	preImage = append(preImage, append(op.Transaction.Data, bytes.Repeat([]byte{0}, 32-len(op.Transaction.Data)%32)...)...) // Right pad to 32-byte increment

	return crypto.Keccak256Hash(preImage), nil
}

//nolint:mnd // Padding to 32 and 64 bytes respectively
func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, err
	}
	mcmsAddress, err := hexToAddress(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to parse MCMS address %q: %w", metadata.MCMAddress, err)
	}

	var preImage []byte
	preImage = append(preImage, mcmDomainSeparatorMetadata...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, chainID), 32)...)
	preImage = append(preImage, mcmsAddress[:]...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, metadata.StartingOpCount), 32)...)
	preImage = append(preImage, common.LeftPadBytes(binary.BigEndian.AppendUint64(nil, metadata.StartingOpCount+e.TxCount), 32)...)
	if e.OverridePreviousRoot {
		preImage = append(preImage, common.LeftPadBytes([]byte{1}, 32)...)
	} else {
		preImage = append(preImage, common.LeftPadBytes([]byte{0}, 32)...)
	}

	return crypto.Keccak256Hash(preImage), nil
}
