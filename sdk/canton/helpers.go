package canton

import (
	"encoding/hex"
	"fmt"
	"strings"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
)

func rawDataFromMCMSTx(newMCMSContractID, newMCMSTemplateID string, rawTx any) map[string]any {
	return map[string]any{
		rawDataKeyNewMCMSContractID: newMCMSContractID,
		rawDataKeyNewMCMSTemplateID: newMCMSTemplateID,
		rawDataKeyRawTx:             rawTx,
	}
}

// transactionResultHash returns an identifier for a Canton ledger submission.
// Prefer the ledger external transaction hash when present; otherwise use commandID.
func transactionResultHash(transaction *apiv2.Transaction, commandID string) string {
	if transaction != nil {
		if ext := transaction.GetExternalTransactionHash(); len(ext) > 0 {
			return "0x" + hex.EncodeToString(ext)
		}
	}

	return commandID
}

func NormalizeTemplateKey(tid string) string {
	tid = strings.TrimPrefix(tid, "#")
	parts := strings.Split(tid, ":")
	if len(parts) < templateIDPartCount {
		return tid
	}

	return parts[len(parts)-2] + ":" + parts[len(parts)-1]
}

// ParseTemplateIDFromString parses a template ID string like "#package:Module:Entity" into its components.
func ParseTemplateIDFromString(templateID string) (packageID, moduleName, entityName string, err error) {
	if !strings.HasPrefix(templateID, "#") {
		return "", "", "", fmt.Errorf("template ID must start with #")
	}
	parts := strings.Split(templateID, ":")
	if len(parts) != templateIDPartCount {
		return "", "", "", fmt.Errorf("template ID must have format #package:module:entity, got: %s", templateID)
	}

	// apiv2.Identifier.PackageId is the raw package id (no leading #).
	return strings.TrimPrefix(parts[0], "#"), parts[1], parts[2], nil
}

// instanceAddressHexEqual reports whether two InstanceAddress hex strings refer to the same address.
func instanceAddressHexEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(a), "0x"), strings.TrimPrefix(strings.TrimSpace(b), "0x"))
}

// FormatTemplateID converts an apiv2.Identifier to a string template ID format.
func FormatTemplateID(id *apiv2.Identifier) string {
	if id == nil {
		return ""
	}

	return id.GetPackageId() + ":" + id.GetModuleName() + ":" + id.GetEntityName()
}
