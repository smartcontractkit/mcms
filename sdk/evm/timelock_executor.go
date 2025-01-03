package evm

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for EVM chains for accessing the RBACTimelock contract
type TimelockExecutor struct {
	client ContractDeployBackend
	auth   *bind.TransactOpts
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client ContractDeployBackend, auth *bind.TransactOpts) *TimelockExecutor {
	return &TimelockExecutor{
		client: client,
		auth:   auth,
	}
}

func (t *TimelockExecutor) Execute(bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash) (string, error) {
	timelock, err := bindings.NewRBACTimelock(common.HexToAddress(timelockAddress), t.client)
	if err != nil {
		return "", err
	}

	calls := make([]bindings.RBACTimelockCall, 0, len(bop.Transactions))
	for i, tx := range bop.Transactions {
		// Unmarshal the AdditionalFields from the operation
		var additionalFields AdditionalFields
		if err = json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return "", err
		}

		calls[i] = bindings.RBACTimelockCall{
			Target: common.HexToAddress(tx.To),
			Data:   tx.Data,
			Value:  additionalFields.Value,
		}
	}

	tx, err := timelock.ExecuteBatch(t.auth, calls, predecessor, salt)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}
