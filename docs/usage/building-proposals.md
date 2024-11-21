# Building Proposals

The library offers 2 ways to build proposals:

## 1. Build Proposal From File

The NewProposal function helps you create a Proposal instance by reading and unmarshaling data from a JSON file. This
guide walks you through using the function to read a proposal from a JSON file, validate it, and create a new Proposal
object.

```go
package main

import (
	"log"
	"os"

	"github.com/smartcontractkit/mcms"
)

func main() {
	// Open the JSON file
	file, err := os.Open("proposal.json")
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	// Create the proposal from the JSON data
	proposal, err := mcms.NewProposal(file)
	if err != nil {
		log.Fatalf("Error creating proposal: %v", err)
	}

	log.Printf("Successfully created proposal: %+v", proposal)
}
```

For the JSON structure of the proposal please check the [MCMS Proposal Format Doc.](/key-concepts/mcm-proposal.md)

## 2. Programmatic Build

The Proposal Builder API provides a fluent interface to construct a Proposal with customizable fields and metadata,
ensuring that each proposal is validated before use.

### Proposal Builder

```go
package main

import (
	"log"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	// Step 1: Initialize the ProposalBuilder
	builder := mcms.NewProposalBuilder()
	selector := types.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector)

	// Step 2: Set Proposal Details
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("Increase staking rewards").
		SetOverridePreviousRoot(false)

	// Step 3: Set Chain Metadata
	builder.SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
		selector: {
			StartingOpCount: 0,
			MCMAddress:      "0x123",
		},
	})

	// append or overwrite chain metadata to the existing map
	builder.AddChainMetadata(selector, types.ChainMetadata{
		StartingOpCount: 0, MCMAddress: "0x345",
	})

	// Step 4: Set Operations
	builder.SetOperations([]types.Operation{
		{
			ChainSelector: selector,
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				To:               "0x1a",
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": 100}`),
			},
		},
		{
			ChainSelector: selector,
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				To:               "0x1b",
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": 200}`),
			},
		},
	})

	// append operations to the existing array
	builder.AddOperation(
		types.Operation{
			ChainSelector: selector,
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				To:               "0x1c",
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": 100}`),
			},
		},
	)

	// Step 5: Build the Proposal
	proposal, err := builder.Build()
	if err != nil {
		log.Fatalf("Error building proposal: %v", err)
	}

	log.Printf("Successfully created proposal: %+v", proposal)
}
```

### Timelock Proposal Builder

The Timelock Proposal Builder is a specialized builder for creating timelock proposals which adds additional builder
methods for setting the action, delay and timelock addresses for the proposal.

```go
package main

import (
	"log"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	// Step 1: Initialize the ProposalBuilder
	builder := mcms.NewTimelockProposalBuilder()
	selector := types.ChainSelector(chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector)

	delay, err := types.ParseDuration("1h")
	if err != nil {
		log.Fatalf("Error parsing duration: %v", err)
	}

	// Step 2: Set Proposal Details
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("Increase staking rewards").
		SetAction(types.TimelockActionSchedule).
		SetDelay(delay).
		SetOverridePreviousRoot(false)

	// Step 3: Set Chain Metadata
	builder.SetChainMetadata(map[types.ChainSelector]types.ChainMetadata{
		selector: {
			StartingOpCount: 0,
			MCMAddress:      "0x123",
		},
	})

	// append or overwrite chain metadata to the existing map
	builder.AddChainMetadata(selector, types.ChainMetadata{
		StartingOpCount: 0, MCMAddress: "0x345",
	})

	// Step 4: Set Timelock addresses
	builder.SetTimelockAddresses(map[types.ChainSelector]string{
		selector: "0x01",
	})

	// append or overwrite timelock addresses to the existing map
	builder.AddTimelockAddress(selector, "0x02")

	// Step 4: Set Operations
	builder.SetOperations([]types.Operation{
		{
			ChainSelector: selector,
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				To:               "0x1a",
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": "100"}`),
			},
		},
		{
			ChainSelector: selector,
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				To:               "0x1b",
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": "200"}`),
			},
		},
	})

	// append operations to the existing array
	builder.AddOperation(
		types.Operation{
			ChainSelector: selector,
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				To:               "0x1c",
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": "100"}`),
			},
		},
	)

	// Step 5: Build the Proposal
	timelockProposal, err := builder.Build()
	if err != nil {
		log.Fatalf("Error building proposal: %v", err)
	}

	log.Printf("Successfully created proposal: %+v", timelockProposal)
}
```
