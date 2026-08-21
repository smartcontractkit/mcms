package stellar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Encoder = (*Encoder)(nil)

type transactionAdditionalFields struct {
	Family          *string `json:"family"`
	EncodingVersion *uint32 `json:"encodingVersion"`
}

func decodeTransactionAdditionalFields(raw json.RawMessage) (transactionAdditionalFields, error) {
	if len(raw) == 0 {
		return transactionAdditionalFields{}, fmt.Errorf("missing Stellar transaction additional fields")
	}
	var fields transactionAdditionalFields
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return transactionAdditionalFields{}, fmt.Errorf("decode Stellar transaction additional fields: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}

		return transactionAdditionalFields{}, fmt.Errorf("decode Stellar transaction additional fields: %w", err)
	}
	if fields.Family == nil || *fields.Family != "stellar" {
		return transactionAdditionalFields{}, fmt.Errorf("invalid Stellar transaction family")
	}
	if fields.EncodingVersion == nil {
		return transactionAdditionalFields{}, fmt.Errorf("missing Stellar transaction encodingVersion")
	}
	if *fields.EncodingVersion != encodingVersion {
		return transactionAdditionalFields{}, fmt.Errorf("%w: %d", ErrUnsupportedEncodingVersion, *fields.EncodingVersion)
	}

	return fields, nil
}

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

	chainID, err := chainNetworkID(e.ChainSelector)
	if err != nil {
		return common.Hash{}, fmt.Errorf("HashOperation: chain id: %w", err)
	}

	multisig, err := parseContractID(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("mcmAddress: %w", err)
	}

	to, err := parseContractID(op.Transaction.To)
	if err != nil {
		return common.Hash{}, fmt.Errorf("transaction.to: %w", err)
	}

	payload, err := DecodeSorobanInvokePayload(op.Transaction.Data)
	if err != nil {
		return common.Hash{}, fmt.Errorf("HashOperation: transaction data: %w", err)
	}
	fields, decErr := decodeTransactionAdditionalFields(op.Transaction.AdditionalFields)
	if decErr != nil {
		return common.Hash{}, fmt.Errorf("HashOperation: additional fields: %w", decErr)
	}
	h, err := HashCurrentStellarOp(
		domainOpStellar,
		chainID,
		[32]byte(multisig),
		uint64(opCount),
		[32]byte(to),
		payload.Function,
		payload.ArgsXDR,
		*fields.EncodingVersion,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("HashOperation: stellar op preimage: %w", err)
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

	chainID, err := chainNetworkID(e.ChainSelector)
	if err != nil {
		return common.Hash{}, fmt.Errorf("HashMetadata: chain id: %w", err)
	}

	multisig, err := parseContractID(metadata.MCMAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("mcmAddress: %w", err)
	}

	configVersion := uint64(1)
	if len(metadata.AdditionalFields) > 0 {
		var fields struct {
			ConfigVersion   *uint64 `json:"configVersion"`
			EncodingVersion *uint32 `json:"encodingVersion"`
		}
		if err := json.Unmarshal(metadata.AdditionalFields, &fields); err != nil {
			return common.Hash{}, fmt.Errorf("HashMetadata: additional fields: %w", err)
		}
		if fields.ConfigVersion != nil {
			configVersion = *fields.ConfigVersion
		}
		if fields.EncodingVersion != nil && *fields.EncodingVersion != encodingVersion {
			return common.Hash{}, fmt.Errorf("%w: %d", ErrUnsupportedEncodingVersion, *fields.EncodingVersion)
		}
	}
	h, err := HashStellarRootMetadata(
		domainMetaStellar,
		chainID,
		multisig,
		metadata.StartingOpCount,
		post,
		e.OverridePreviousRoot,
		configVersion,
		encodingVersion,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("HashMetadata: root metadata preimage: %w", err)
	}

	return h, nil
}
