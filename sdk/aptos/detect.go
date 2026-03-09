package aptos

import (
	"encoding/json"

	"github.com/smartcontractkit/mcms/types"
)

const curseMCMSPackageName = "curse_mcms"

// MCMSTypeFromOperations inspects the Aptos transactions for the given chain
// selector and returns MCMSTypeCurse when any transaction has
// package_name == "curse_mcms" in its additional fields. Otherwise it returns
// MCMSTypeRegular (the zero value).
func MCMSTypeFromOperations(ops []types.BatchOperation, cs types.ChainSelector) MCMSType {
	for _, batchOp := range ops {
		if batchOp.ChainSelector != cs {
			continue
		}
		for _, tx := range batchOp.Transactions {
			var af AdditionalFields
			if err := json.Unmarshal(tx.AdditionalFields, &af); err == nil {
				if af.PackageName == curseMCMSPackageName {
					return MCMSTypeCurse
				}
			}
		}
	}

	return MCMSTypeRegular
}
