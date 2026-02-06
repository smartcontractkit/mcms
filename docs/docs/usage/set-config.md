# Setting Configuration on MCMs Contract

You can use the `Configurer` of each chain family's SDK to set the config of the contract:

```go
package main

import (
  "context"
  "log"

  "github.com/ethereum/go-ethereum/accounts/abi/bind"
  "github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
  "github.com/ethereum/go-ethereum/common"
  "github.com/gagliardetto/solana-go"
  rpc2 "github.com/gagliardetto/solana-go/rpc"
  chainsel "github.com/smartcontractkit/chain-selectors"

  "github.com/smartcontractkit/mcms/sdk/evm"
  mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
  "github.com/smartcontractkit/mcms/types"
)

func main() {
  config := types.Config{
    Quorum: 2,
    Signers: []common.Address{
      common.HexToAddress("0x123"),
      common.HexToAddress("0x456"),
      common.HexToAddress("0x789"),
    },
    GroupSigners: []types.Config{
      {
        Quorum: 5,
        Signers: []common.Address{
          common.HexToAddress("0x123"),
          common.HexToAddress("0x456"),
          common.HexToAddress("0x789"),
          common.HexToAddress("0xabc"),
          common.HexToAddress("0xdef"),
        },
      },
    },
  }
  ctx := context.Background()

  // On EVM
  solanaSelector := chainsel.SOLANA_DEVNET.Selector
  mcmsContractAddr := "0x123"
  backend := backends.SimulatedBackend{}
  auth := &bind.TransactOpts{}
  configurerEVM := evm.NewConfigurer(backend, auth)
  tx, err := configurerEVM.SetConfig(ctx, mcmsContractAddr, &config, false)
  if err != nil {
    log.Fatalf("failed to set config: %v", err)
  }
  log.Printf("set config evm tx hash: %s\n", tx.Hash)

  // On Solana
  mcmsContractID := "6UmMZr5MEqiKWD5jqTJd1WCR5kT8oZuFYBLJFi1o6GQX"
  rpc := rpc2.New("https://api.devnet.solana.com")
  wallet := solana.NewWallet()
  configurerSolana := mcmsSolana.NewConfigurer(rpc, wallet.PrivateKey, types.ChainSelector(solanaSelector))
  tx, err = configurerSolana.SetConfig(ctx, mcmsContractID, &config, false)
  if err != nil {
    log.Fatalf("failed to set config: %v", err)
  }
  log.Printf("set config solana tx hash: %s\n", tx.Hash)
}

```
