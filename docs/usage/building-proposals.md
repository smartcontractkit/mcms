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
	proposal, err := NewProposal(file)
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
	"fmt"

	"mcms"

	"github.com/smartcontractkit/mcms/types"
)

func main() {
	// Step 1: Initialize the ProposalBuilder
	builder := mcms.NewProposalBuilder()

	// Step 2: Set Proposal Details
	builder.
		SetVersion("1.0").
		SetValidUntil(1700000000).
		SetDescription("Increase staking rewards").
		AddSignature(types.Signature{Signer: "0x123", Signature: "abcd1234"}).
		SetOverridePreviousRoot(false).
		UseSimulatedBackend(true)

	// Step 3: Add Chain Metadata
	builder.AddChainMetadata(types.ChainSelector{ChainID: "0x1"}, types.ChainMetadata{Data: "example"})
	// Or set the full metadata map
	chainMetadataMap := map[types.ChainSelector]types.ChainMetadata{
		{ChainID: "0x1"}: {Data: "Ethereum Mainnet Metadata"},
		{ChainID: "0x2"}: {Data: "BSC Mainnet Metadata"},
	}
	builder.SetChainMetadata(chainMetadataMap)

	// Step 4: Add Transactions
	builder.AddTransaction(types.Operation{To: "0x5678", Data: "0xabcdef"})
	// Or set Full Transactions List
	transactions := []types.ChainOperation{
		{Target: "0xABCDEF", Value: "500", Data: "0xdata1"},
		{Target: "0x123456", Value: "300", Data: "0xdata2"},
	}
	builder.SetTransactions(transactions)

	// Step 5: Build the Proposal
	proposal, err := builder.Build()
	if err != nil {
		fmt.Println("Failed to build proposal:", err)
		return
	}

	fmt.Println("Proposal created:", proposal)
}
```