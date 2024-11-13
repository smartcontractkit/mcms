# Building Proposals

The library offers 2 ways to build proposals:

## 1. Build Proposal From File

The NewProposal function helps you create a Proposal instance by reading and
unmarshaling data from a JSON file. This guide walks you through using the
function to read a proposal from a JSON file, validate it, and create a new Proposal
object.

```golang
package main

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/mcms"
)

func main() {
	// Open the JSON file
	file, err := os.Open("proposal.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create the proposal from the JSON data
	proposal, err := mcms.NewProposal(file)
	if err != nil {
		fmt.Println("Error creating proposal:", err)
		return
	}

	fmt.Println("Successfully created proposal:", proposal)
}
```

For the JSON structure of the proposal please check the [MCMS Proposal Format Doc.](../key-concepts/mcms-proposal.md)

## 2. Programmatic Build

The Proposal Builder API provides a fluent interface to construct a Proposal with
customizable fields and metadata, ensuring that each proposal is validated before use.

```golang
package main

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
		log.Fatalf("Error opening proposal: %v", err)
	}
	defer file.Close()
	proposal, err := mcms.NewProposal(file)
	if err != nil {
		log.Fatalf("Error loading proposal: %v", err)
	}

	// Step 2: Initialize the Chain Family Executors
	selector1 := types.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector)
	selector2 := types.ChainSelector(chain_selectors.ETHEREUM_TESTNET_GOERLI_ARBITRUM_1.Selector)
	backend1 := backends.SimulatedBackend{}
	backend2 := backends.SimulatedBackend{}
	executor1 := evm.NewEVMExecutor(evm.NewEVMEncoder(0, uint64(selector1), false), backend1, nil)
	executor2 := evm.NewEVMExecutor(evm.NewEVMEncoder(0, uint64(selector2), false), backend2, nil)
	executorsMap := map[types.ChainSelector]sdk.Executor{
		selector1: executor1,
		selector2: executor2,
	}
	// Step 3: Create the chain MCMS proposal executor
	executable, err := mcms.NewExecutable(proposal, executorsMap)
	if err != nil {
		log.Fatalf("Error creating executable: %v", err)
	}

	// Step 4: SetRoot on all chains
	for _, selector := range []types.ChainSelector{selector1, selector2} {
		_, err = executable.SetRoot(selector)
		if err != nil {
			log.Fatalf("Error setting root: %v", err)
		}
	}
	// Step 5: Execute the all the operations by looping through the proposal
	for idx := range proposal.Operations {
		_, err = executable.Execute(idx)
		if err != nil {
			log.Fatalf("Error executing operation: %v", err)
		}

	}

}
```