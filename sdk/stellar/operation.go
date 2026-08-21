package stellar

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/types"
)

// NewTransaction creates the canonical generic MCMS transaction for a Soroban
// invocation. The target and invocation payload remain in the generic
// Transaction fields; AdditionalFields contains only family/version metadata.
func NewTransaction(target, function string, args []xdr.ScVal, contractType string, tags []string) (types.Transaction, error) {
	if _, err := parseContractID(target); err != nil {
		return types.Transaction{}, fmt.Errorf("stellar target: %w", err)
	}
	data, err := EncodeSorobanInvokePayload(function, args)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("stellar invoke payload: %w", err)
	}

	additionalFields, err := json.Marshal(struct {
		Family          string `json:"family"`
		EncodingVersion uint32 `json:"encodingVersion"`
	}{Family: chainsel.FamilyStellar, EncodingVersion: encodingVersion})
	if err != nil {
		return types.Transaction{}, fmt.Errorf("encode Stellar transaction metadata: %w", err)
	}

	return types.Transaction{
		OperationMetadata: types.OperationMetadata{
			ContractType: contractType,
			Tags:         tags,
		},
		To:               target,
		Data:             data,
		AdditionalFields: additionalFields,
	}, nil
}

// NewBatchOperation creates a one-transaction Stellar MCMS batch.
func NewBatchOperation(chainSelector types.ChainSelector, target, function string, args []xdr.ScVal, contractType string, tags []string) (types.BatchOperation, error) {
	tx, err := NewTransaction(target, function, args, contractType, tags)
	if err != nil {
		return types.BatchOperation{}, err
	}

	return types.BatchOperation{ChainSelector: chainSelector, Transactions: []types.Transaction{tx}}, nil
}

// IsAddress keeps address validation available to integration adapters without
// exposing the internal contract-ID representation used by the encoder.
func IsAddress(address string) bool {
	_, err := parseContractID(address)

	return err == nil
}

// AddressHash returns the raw 32-byte contract identifier for callers that
// need to compare Stellar target addresses.
func AddressHash(address string) (common.Hash, error) {
	id, err := parseContractID(address)
	if err != nil {
		return common.Hash{}, err
	}

	return common.Hash(id), nil
}
