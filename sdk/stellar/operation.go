package stellar

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// ConfigToSetConfigInputs flattens a nested MCMS [types.Config] tree into the
// canonical Stellar set_config / initialize parameter types. Signer addresses are
// left-padded to 32 bytes (Solidity address layout).
func ConfigToSetConfigInputs(cfg *types.Config) (mcmsbindings.SignerAddresses, mcmsbindings.SignerGroups, [32]byte, [32]byte, error) {
	groupQuorums, groupParents, signerAddrs, signerGroups, err := sdk.ExtractSetConfigInputs(cfg)
	if err != nil {
		return mcmsbindings.SignerAddresses{}, mcmsbindings.SignerGroups{}, [32]byte{}, [32]byte{}, err
	}

	addresses := make([][32]byte, len(signerAddrs))
	for i, addr := range signerAddrs {
		copy(addresses[i][12:], addr.Bytes())
	}

	groups := make([]uint32, len(signerGroups))
	for i, g := range signerGroups {
		groups[i] = uint32(g)
	}

	var gq, gp [32]byte
	for i := 0; i < 32; i++ {
		gq[i] = groupQuorums[i]
		gp[i] = groupParents[i]
	}

	return mcmsbindings.SignerAddresses{Inner: addresses},
		mcmsbindings.SignerGroups{Inner: groups},
		gq, gp, nil
}

// NewSetConfigOperation creates the canonical Stellar MCMS set_config
// operation. It is used when configuration must be submitted through an
// MCMS proposal instead of directly by the deployer.
func NewSetConfigOperation(target string, cfg *types.Config, clearRoot bool, contractType string, tags []string) (types.Transaction, error) {
	signerAddresses, signerGroups, groupQuorums, groupParents, err := ConfigToSetConfigInputs(cfg)
	if err != nil {
		return types.Transaction{}, fmt.Errorf("stellar set_config inputs: %w", err)
	}

	addresses, err := signerAddresses.ToScVal()
	if err != nil {
		return types.Transaction{}, fmt.Errorf("stellar signer addresses: %w", err)
	}
	groups, err := signerGroups.ToScVal()
	if err != nil {
		return types.Transaction{}, fmt.Errorf("stellar signer groups: %w", err)
	}

	args := []xdr.ScVal{
		addresses,
		groups,
		scval.Bytes32ToScVal(groupQuorums),
		scval.Bytes32ToScVal(groupParents),
		scval.BoolToScVal(clearRoot),
	}

	return NewTransaction(target, "set_config", args, contractType, tags)
}

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
	}{Family: "stellar", EncodingVersion: encodingVersion})
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
