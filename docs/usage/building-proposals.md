# Building Proposals

### Table of Contents

- [1. Build Proposal From File](#1-build-proposal-from-file)
- [2. Programmatic Build](#2-programmatic-build)
  - [Proposal Builder](#proposal-builder)
  - [Timelock Proposal Builder](#timelock-proposal-builder)
- [Adding Chain Specific Operations to Proposal](#adding-chain-specific-operations-to-proposal)
  - EVM Operations
  - Solana Operations

## 1. Build Proposal From File

The NewProposal function helps you create a Proposal instance by reading and unmarshaling data from a JSON file. This
guide walks you through using the function to read a proposal from a JSON file, validate it, and create a new Proposal
object.

```go
package main

import (
  "log"
  "os"
  "io"

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

### Build Proposal Given Staged but Non-Executed Predecessor Proposals

In scenarios where a proposal is generated with the assumption that multiple proposals are executed beforehand, 
you can enable proposals to be signed in parallel with a pre-determined execution order. This can be achieved
by passing a list of files using the `WithPredecessors` functional option, as shown below:

```go
package main

import (
  "log"
  "os"
  "io"

  "github.com/smartcontractkit/mcms"
)

func main() {
  // Open the JSON file for the new proposal
  file, err := os.Open("proposal.json")
  if err != nil {
    log.Fatalf("Error opening file: %v", err)
  }
  defer file.Close()

  // Open the JSON file for the predecessor proposal
  preFile, err := os.Open("pre-proposal.json")
  if err != nil {
    log.Fatalf("Error opening predecessor file: %v", err)
  }
  defer preFile.Close()

  // Create the proposal from the JSON data
  proposal, err := mcms.NewProposal(file, mcms.WithPredecessors([]io.Reader{preFile}))
  if err != nil {
    log.Fatalf("Error creating proposal: %v", err)
  }

  log.Printf("Successfully created proposal: %+v", proposal)
}
```

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

# Adding Chain Specific Operations to Proposal

The mcms lib provides helper functionality to easily build chain
specific operations and add them to a proposal. Here are some examples:

### EVM Operations

Use the `evm.NewTransaction` helper to build EVM specific transaction.

```go
// Create an evm specific tx for the proposal operation
tx := evm.NewTransaction(
common.Address{},
[]byte("data bytes of the transaction"),
5, // Value in GWEI
"MyEVMContractType",
[]string{"tag1", "tag2"},
)

op := builder.AddOperation(types.Operation{
ChainSelector: selector,
Transaction:   tx,
})
```

### Solana Operations

Use the `solana.NewTransaction` helper to build Solana specific transaction.

```go
// Create a solana specific tx for the proposal operation
accounts := []*solana.AccountMeta{
{
PublicKey:  solana.MustPublicKeyFromBase58("account pub key"),
IsSigner:   false,
IsWritable: true,
}
}

tx := solana.NewTransaction(
"programIDGoesHere",
[]byte("data bytes of the instruction"),
accounts,
"MySolanaContractType",
[]string{"tag1", "tag2"}
)

op := builder.AddOperation(types.Operation{ChainSelector: selector, Transaction: tx})

```
