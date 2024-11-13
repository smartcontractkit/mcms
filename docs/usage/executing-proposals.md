# Executing Proposals

Here is an example of how to run SetRoot and Execute on a signed proposal.

```go
package examples

import (
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	// Step 1: Load the Proposal
	file, err := os.Open("proposal.json")
	if err != nil {
		log.Fatalf("Error opening proposal file: %v", err)
		return
	}
	defer file.Close()
	proposal, err := mcms.NewProposal(file)
	if err != nil {
		log.Fatalf("Error opening proposal file: %v", err)
		return
	}

	// Step 2: Initialize the Chain Family Executors
	selector := chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector
	backend := backends.SimulatedBackend{}
	executor := evm.NewEVMExecutor(evm.NewEVMEncoder(0, selector, false), backend, nil)
	executorsMap := map[types.ChainSelector]sdk.Executor{
		types.ChainSelector(selector): executor,
	}
	// Step 3: Create the chain MCMS proposal executor
	executable, err := mcms.NewExecutable(proposal, executorsMap)
	if err != nil {
		log.Fatalf("Error opening proposal file: %v", err)
	}

	// Step 4: SetRoot of a proposal
	_, err = executable.SetRoot(types.ChainSelector(selector))
	if err != nil {
		log.Fatalf("Error opening proposal file: %v", err)
	}

	// Step 5: Execute the first operation of the proposal.
	_, err = executable.Execute(0)
	if err != nil {
		log.Fatalf("Error opening proposal file: %v", err)
	}
}


```