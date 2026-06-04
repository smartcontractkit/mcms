package canton

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/types"
)

// defaultMCMSInstanceIDCandidates are common Canton MCMS instance IDs tried when inferring
// chain metadata from mcmAddress + party (e.g. proposals generated without additionalFields).
var defaultMCMSInstanceIDCandidates = []string{mcmsInstanceIDCCIP, mcmsInstanceIDCCV, mcmsInstanceIDDefault}

// EnsureChainMetadata fills Canton-specific additionalFields when they are missing or incomplete.
func EnsureChainMetadata(
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	action types.TimelockAction,
) (types.ChainMetadata, error) {
	fields, err := resolveAdditionalFieldsMetadata(metadata, bop, action)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	additionalFieldsBytes, err := json.Marshal(fields)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("marshal canton additional fields: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount:  metadata.StartingOpCount,
		MCMAddress:       metadata.MCMAddress,
		AdditionalFields: additionalFieldsBytes,
	}, nil
}

func resolveAdditionalFieldsMetadata(
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	action types.TimelockAction,
) (AdditionalFieldsMetadata, error) {
	if len(metadata.AdditionalFields) > 0 {
		var fields AdditionalFieldsMetadata
		if err := json.Unmarshal(metadata.AdditionalFields, &fields); err != nil {
			return AdditionalFieldsMetadata{}, fmt.Errorf("unmarshal metadata additional fields: %w", err)
		}
		if err := fields.Validate(); err == nil {
			return fields, nil
		}
	}

	party, err := partyFromBatchOperation(bop)
	if err != nil {
		return AdditionalFieldsMetadata{}, err
	}

	instanceID, err := resolveInstanceID(metadata.MCMAddress, party, defaultMCMSInstanceIDCandidates)
	if err != nil {
		return AdditionalFieldsMetadata{}, err
	}

	role, err := CantonRoleFromAction(action)
	if err != nil {
		return AdditionalFieldsMetadata{}, fmt.Errorf("canton role from action: %w", err)
	}
	multisigID := fmt.Sprintf("%s@%s-%s", instanceID, party, strings.ToLower(role.String()))

	fields := AdditionalFieldsMetadata{
		ChainId:    defaultCantonChainID,
		MultisigId: multisigID,
		InstanceId: instanceID,
	}
	if err := fields.Validate(); err != nil {
		return AdditionalFieldsMetadata{}, fmt.Errorf("inferred canton additional fields invalid: %w", err)
	}

	return fields, nil
}

func partyFromBatchOperation(bop types.BatchOperation) (string, error) {
	for _, tx := range bop.Transactions {
		if len(tx.AdditionalFields) == 0 {
			continue
		}
		var af AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &af); err != nil {
			continue
		}
		if party := partyFromRawInstanceAddress(af.TargetInstanceAddress); party != "" {
			return party, nil
		}
	}
	if len(bop.Transactions) > 0 && bop.Transactions[0].To != "" {
		if party := partyFromRawInstanceAddress(bop.Transactions[0].To); party != "" {
			return party, nil
		}
	}

	return "", fmt.Errorf("unable to infer Canton party from batch operation transactions")
}

func partyFromRawInstanceAddress(raw string) string {
	at := strings.Index(raw, "@")
	if at <= 0 || at >= len(raw)-1 {
		return ""
	}

	return raw[at+1:]
}

func resolveInstanceID(mcmAddressHex, party string, candidates []string) (string, error) {
	target := strings.ToLower(strings.TrimPrefix(mcmAddressHex, "0x"))
	for _, candidate := range candidates {
		raw := candidate + "@" + party
		hash := crypto.Keccak256([]byte(raw))
		if strings.EqualFold(hex.EncodeToString(hash), target) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf(
		"unable to infer MCMS instanceId for address %s and party %s (tried %v)",
		mcmAddressHex, party, candidates,
	)
}
