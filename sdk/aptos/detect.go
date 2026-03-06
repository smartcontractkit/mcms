package aptos

import (
	"encoding/json"

	"github.com/smartcontractkit/mcms/types"
)

const curseMCMSPackageName = "curse_mcms"

// IsCurseMCMSFromOperations returns true if any Aptos transaction for the
// given chain selector has package_name == "curse_mcms" in its additional
// fields. This is a reliable self-describing signal already present in every
// CurseMCMS proposal.
func IsCurseMCMSFromOperations(ops []types.BatchOperation, cs types.ChainSelector) bool {
	for _, batchOp := range ops {
		if batchOp.ChainSelector != cs {
			continue
		}
		for _, tx := range batchOp.Transactions {
			var af AdditionalFields
			if err := json.Unmarshal(tx.AdditionalFields, &af); err == nil {
				if af.PackageName == curseMCMSPackageName {
					return true
				}
			}
		}
	}

	return false
}
