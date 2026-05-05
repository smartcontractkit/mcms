package stellar

import (
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-stellar/bindings"

	"github.com/smartcontractkit/mcms/types"
)

// txHashInvoker is optionally implemented by invokers that expose the last submitted Soroban tx hash.
type txHashInvoker interface {
	LastSubmittedTransactionHash() string
}

func stellarTransactionResult(invoker bindings.Invoker) types.TransactionResult {
	hash := ""

	if th, ok := invoker.(txHashInvoker); ok {
		hash = th.LastSubmittedTransactionHash()
	}

	return types.NewTransactionResult(hash, nil, chainsel.FamilyStellar)
}
