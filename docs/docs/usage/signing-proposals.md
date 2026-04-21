# Signing Proposals

For signing proposals, we use the methods that come with the `Proposal` type.

```go
package examples

import (
  "crypto/ecdsa"
  "fmt"
  "log"
  "os"

  "github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
  "github.com/gagliardetto/solana-go/rpc"
  chainsel "github.com/smartcontractkit/chain-selectors"

  "github.com/smartcontractkit/mcms"
  "github.com/smartcontractkit/mcms/sdk"
  "github.com/smartcontractkit/mcms/sdk/evm"
  "github.com/smartcontractkit/mcms/sdk/solana"
  "github.com/smartcontractkit/mcms/types"
)

func main() {
  file, err := os.Open("proposal.json")
  if err != nil {
    log.Fatalf("failed to open file: %v", err)
  }
  defer file.Close()

  // 1. Create the proposal from the JSON data
  proposal, err := mcms.NewProposal(file)
  if err != nil {
    log.Fatalf("failed to open file: %v", err)
  }

  // 2. Create the signable type from the proposal
  selector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector

  // if evm required: Add EVM Inspector
  backend := backends.SimulatedBackend{}
  inspectorsMap := make(map[types.ChainSelector]sdk.Inspector)
  inspectorsMap[types.ChainSelector(selector)] = evm.NewInspector(backend)

  // if solana required: Add Solana Inspector
  client := rpc.New("https://api.devnet.solana.com")
  inspectorsMap[types.ChainSelector(chainsel.SOLANA_DEVNET.Selector)] = solana.NewInspector(client)

  // Create Signable
  signable, err := mcms.NewSignable(proposal, inspectorsMap)

  // 3. Sign the proposal bytes
  // This will be done via ledger, using a private key KMS, etc.
  // For the sake of this example, we will generate a signature using an empty private key
  signer := mcms.NewPrivateKeySigner(&ecdsa.PrivateKey{})
  // Or using ledger, you can call NewLedgerSigner and provide the derivation path as a parameter
  // signerLedger := mcms.NewLedgerSigner([]uint32{44, 60, 0, 0, 0})
  signature, err := signable.Sign(signer)
  if err != nil {
    log.Fatalf("failed to open file: %v", err)
  }

  /// 4. Add the signature
  proposal.AppendSignature(signature)
  fmt.Println("Successfully created proposal:", proposal)
}

```
