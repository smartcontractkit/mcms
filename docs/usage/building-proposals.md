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
package examples

import (
	"fmt"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	// Step 1: Initialize the ProposalBuilder
	builder := mcms.NewProposalBuilder()
	selector := chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector
	// Step 2: Set Proposal Details
	builder.
		SetVersion("v1").
		SetValidUntil(1794610529).
		SetDescription("Increase staking rewards").
		AddSignature(types.Signature{}). // For details on signature generation see https://github.com/smartcontractkit/mcms/blob/main/docs/usage/signing-proposals.md
		SetOverridePreviousRoot(false).
		UseSimulatedBackend(true)

	// Step 3: Add Chain Metadata
	builder.AddChainMetadata(types.ChainSelector(selector), types.ChainMetadata{StartingOpCount: 0, MCMAddress: "0x123"})
	// Or set the full metadata map
	chainMetadataMap := map[types.ChainSelector]types.ChainMetadata{
		// Each entry sets the timelock address on the given chain selector
		types.ChainSelector(selector): {StartingOpCount: 0, MCMAddress: "0x123"},
	}
	builder.SetChainMetadata(chainMetadataMap)

	// Step 4: Add Transactions
	builder.AddOperation(
		types.Operation{
			ChainSelector: types.ChainSelector(selector),
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": "100"}`), // Chain specific fields for the operation
			},
		})
	// Or set Full Transactions List
	transactions := []types.Operation{
		{
			ChainSelector: types.ChainSelector(selector),
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": "100"}`), // Chain specific fields for the operation
			},
		},
		{
			ChainSelector: types.ChainSelector(selector),
			Transaction: types.Transaction{
				OperationMetadata: types.OperationMetadata{
					ContractType: "some-contract",
					Tags:         []string{"staking", "rewards"},
				},
				Data:             []byte("data bytes of the transaction"),
				AdditionalFields: []byte(`{"value": "200"}`), // Chain specific fields for the operation
			},
		},
	}
	builder.SetOperations(transactions)

	// Step 5: Build the Proposal
	proposal, err := builder.Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("Proposal created:", proposal)
}

```