package stellar

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/types"
)

// OperationID is not implemented for Stellar timelock flows until timelock parity exists on Soroban.
func OperationID(
	_ types.BatchOperation,
	_ types.TimelockAction,
	_ common.Hash,
	_ common.Hash,
) (common.Hash, error) {
	return common.Hash{}, fmt.Errorf("stellar timelock OperationID is not implemented")
}
