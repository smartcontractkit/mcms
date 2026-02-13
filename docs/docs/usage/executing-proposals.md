# Executing Proposals

Here is an example of how to run SetRoot and Execute on a signed proposal.

```go
package examples

import (
  "context"
  "log"
  "os"

  "github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
  "github.com/gagliardetto/solana-go"
  "github.com/gagliardetto/solana-go/rpc"
  chainsel "github.com/smartcontractkit/chain-selectors"

  "github.com/smartcontractkit/mcms"
  "github.com/smartcontractkit/mcms/sdk"
  "github.com/smartcontractkit/mcms/sdk/evm"
  mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
  "github.com/smartcontractkit/mcms/types"
)

func main() {
  // Step 1: Load the Proposal
  ctx := context.Background()
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
  evmSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
  solanaSelector := chainsel.SOLANA_DEVNET.Selector

  // EVM executor
  backend := backends.SimulatedBackend{}
  evmExecutor := evm.NewExecutor(evm.NewEncoder(types.ChainSelector(evmSelector), 0, false, false), backend, nil)

  // Solana executor
  client := rpc.New("https://api.devnet.solana.com")
  solanaKey, err := solana.NewRandomPrivateKey()
  if err != nil {
    log.Fatalf("Error creating solana key: %v", err)
    return
  }

  encoder := mcmsSolana.NewEncoder(types.ChainSelector(solanaSelector), 0, false)
  solanaExecutor := mcmsSolana.NewExecutor(client, solanaKey, encoder)

  // Build executors map
  executorsMap := map[types.ChainSelector]sdk.Executor{
    types.ChainSelector(evmSelector):    evmExecutor,
    types.ChainSelector(solanaSelector): solanaExecutor,
  }
  // Step 3: Create the chain MCMS proposal  executor
  executable, err := mcms.NewExecutable(proposal, executorsMap)
  if err != nil {
    log.Fatalf("Error opening proposal file: %v", err)
  }

  // Step 4: SetRoot of a proposal
  // On EVM
  _, err = executable.SetRoot(ctx, types.ChainSelector(evmSelector))
  if err != nil {
    log.Fatalf("Error calling set root: %v", err)
  }
  // On Solana
  _, err = executable.SetRoot(ctx, types.ChainSelector(solanaSelector))
  if err != nil {
    log.Fatalf("Error calling set root: %v", err)
  }

  // Step 5: Execute the first operation of the proposal.
  _, err = executable.Execute(ctx, 0)
  if err != nil {
    log.Fatalf("Error calling execute: %v", err)
  }

}


```
