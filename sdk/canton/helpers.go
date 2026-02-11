package canton

import (
	"fmt"
	"strings"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
)

const (
	MCMSTemplateKey = "MCMS.Main:MCMS"
)

func NormalizeTemplateKey(tid string) string {
	tid = strings.TrimPrefix(tid, "#")
	parts := strings.Split(tid, ":")
	if len(parts) < 3 {
		return tid
	}

	return parts[len(parts)-2] + ":" + parts[len(parts)-1]
}

// parseTemplateIDFromString parses a template ID string like "#package:Module:Entity" into its components
func parseTemplateIDFromString(templateID string) (packageID, moduleName, entityName string, err error) {
	if !strings.HasPrefix(templateID, "#") {
		return "", "", "", fmt.Errorf("template ID must start with #")
	}
	parts := strings.Split(templateID, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("template ID must have format #package:module:entity, got: %s", templateID)
	}

	return parts[0], parts[1], parts[2], nil
}

// ParseTemplateIDFromString is the exported version of parseTemplateIDFromString
func ParseTemplateIDFromString(templateID string) (packageID, moduleName, entityName string, err error) {
	return parseTemplateIDFromString(templateID)
}

// formatTemplateID converts an apiv2.Identifier to a string template ID format
func formatTemplateID(id *apiv2.Identifier) string {
	if id == nil {
		return ""
	}
	return id.GetPackageId() + ":" + id.GetModuleName() + ":" + id.GetEntityName()
}

// FormatTemplateID is the exported version of formatTemplateID
func FormatTemplateID(id *apiv2.Identifier) string {
	return formatTemplateID(id)
}
