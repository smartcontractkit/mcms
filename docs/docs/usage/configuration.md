# Transforming Configurations

Each chain family might have a different structure required for the contract configuration.
The MCMS lib allows you to transform from a chain agnostic configuration to the chain specific
configuration structures using the `ConfigTransformer` types.

```go
package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

func main() {
	// This example will use the ConfigTransformer to get the EVM chain-specific config
	evmTransformer := evm.NewConfigTransformer()
	// Step 1: Let's create a config of 5 signers with a quorum of 3 and no nested subgroups.
	config, err := types.NewConfig(
		3,
		[]common.Address{
			common.HexToAddress("0x01"),
			common.HexToAddress("0x02"),
			common.HexToAddress("0x03"),
			common.HexToAddress("0x04"),
			common.HexToAddress("0x05"),
		},
		[]types.Config{})
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}
	// Step 2: Convert the chain-agnostic config to the chain-specific config
	evmConfig, err := evmTransformer.ToChainConfig(config)
	if err != nil {
		log.Fatalf("Failed to convert config: %v", err)
	}
	fmt.Println("EVM Config: %+v", evmConfig)

	// Step 3: Convert the chain-specific config back to the chain-agnostic config
	chainAgnosticConfig, err := evmTransformer.ToConfig(evmConfig)
	if err != nil {
		log.Fatalf("Failed to convert config: %v", err)
	}
	fmt.Println("Chain Agnostic Config: %+v", chainAgnosticConfig)

}


```