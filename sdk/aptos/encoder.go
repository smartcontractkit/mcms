package aptos

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
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

func (e *Encoder) HashOperation(opCount uint32, metadata types.ChainMetadata, op types.Operation) (common.Hash, error) {
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, err
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)
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
	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err := json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return common.Hash{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

	ser := bcs.Serializer{}
	ser.FixedBytes(mcmDomainSeparatorOp)
	ser.U8(uint8(additionalFieldsMetadata.Role))
	ser.U256(*chainIDBig)
	ser.Struct(&mcmsAddress)
	ser.U64(uint64(opCount))
	ser.Struct(&toAddress)
	ser.WriteString(additionalFields.ModuleName)
	ser.WriteString(additionalFields.Function)
	ser.WriteBytes(op.Transaction.Data)

	return crypto.Keccak256Hash(ser.ToBytes()), nil
}

func (e *Encoder) HashMetadata(metadata types.ChainMetadata) (common.Hash, error) {
	chainID, err := chain_selectors.AptosChainIdFromSelector(uint64(e.ChainSelector))
	if err != nil {
		return common.Hash{}, err
	}
	chainIDBig := (&big.Int{}).SetUint64(chainID)
	mcmsAddress, err := hexToAddress(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to parse MCMS address %q: %w", metadata.MCMAddress, err)
	}
	var additionalFieldsMetadata AdditionalFieldsMetadata
	if len(metadata.AdditionalFields) > 0 {
		if err = json.Unmarshal(metadata.AdditionalFields, &additionalFieldsMetadata); err != nil {
			return common.Hash{}, fmt.Errorf("failed to unmarshal additional fields metadata: %w", err)
		}
	}

	ser := bcs.Serializer{}
	ser.FixedBytes(mcmDomainSeparatorMetadata)
	ser.U8(uint8(additionalFieldsMetadata.Role))
	ser.U256(*chainIDBig)
	ser.Struct(&mcmsAddress)
	ser.U64(metadata.StartingOpCount)
	ser.U64(metadata.StartingOpCount + e.TxCount)
	ser.Bool(e.OverridePreviousRoot)

	return crypto.Keccak256Hash(ser.ToBytes()), nil
}
