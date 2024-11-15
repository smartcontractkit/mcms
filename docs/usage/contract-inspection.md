# Contract inspection

You can use the chain specific SDKs to inspect the state of your MCMS and timelock contracts.

### Example: MCMS Inspection

```go
package main

import (
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"

	"github.com/smartcontractkit/mcms/sdk/evm"
)

func main() {
	// Inspecting MCMS contract on EVM
	backend := backends.SimulatedBackend{}
	inspector := evm.NewInspector(backend)
	contractAddress := "0x123" // replace with your address

	// Get the op count
	opcount, err := inspector.GetOpCount(contractAddress)
	if err != nil {
		log.Fatalf("failed to get op count: %v", err)
	}
	log.Printf("Op count: %d", opcount)

	// Get the config
	config, err := inspector.GetConfig(contractAddress)
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}
	log.Printf("Config: %+v", config)

	// Get the root
	root, validUntil, err := inspector.GetRoot(contractAddress)
	if err != nil {
		log.Fatalf("failed to get root: %v", err)
	}
	log.Printf("Root: %s, Valid until: %d", root.Hex(), validUntil)

	// Get the proposers
	metadata, err := inspector.GetRootMetadata(contractAddress)
	if err != nil {
		log.Fatalf("failed to get root metadata: %v", err)
	}
	log.Printf("Metadata: %+v", metadata)

}
```

### Example: Timelock Inspection

```go
package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"

	"github.com/smartcontractkit/mcms/sdk/evm"
)

func main() {
	// Inspecting Timelock contract on EVM
	backend := backends.SimulatedBackend{}
	inspector := evm.NewTimelockInspector(backend)
	contractAddress := "0x123" // replace with your address

	// Get the proposers
	proposers, err := inspector.GetProposers(contractAddress)
	if err != nil {
		log.Fatalf("failed to get op count: %v", err)
	}
	log.Printf("proposers: %d", proposers)

	// Get the bypassers
	bypassers, err := inspector.GetBypassers(contractAddress)
	if err != nil {
		log.Fatalf("failed to get bypassers: %v", err)
	}
	log.Printf("bypassers: %+v", bypassers)

	// Get the executors
	executors, err := inspector.GetExecutors(contractAddress)
	if err != nil {
		log.Fatalf("failed to get executors: %v", err)
	}
	log.Printf("executors: %s", executors)

	// Get the cancellers
	cancellers, err := inspector.GetCancellers(contractAddress)
	if err != nil {
		log.Fatalf("failed to get root metadata: %v", err)
	}
	log.Printf("Metadata: %+v", cancellers)

	// Get operation statuses, opID is a [32]byte representing the operation ID
	opID := [32]byte{} // replace with your operation ID
	isOp, err := inspector.IsOperation(contractAddress, opID)
	if err != nil {
		log.Fatalf("failed to get operation status: %v", err)
	}
	log.Printf("IsOperation: %t", isOp)

	isReady, err := inspector.IsOperationReady(contractAddress, opID)
	if err != nil {
		log.Fatalf("failed to get operation status: %v", err)
	}
	fmt.Printf("IsOperationReady: %t", isReady)

	isPending, err := inspector.IsOperationPending(contractAddress, opID)
	if err != nil {
		log.Fatalf("failed to get operation status: %v", err)
	}
	fmt.Printf("IsOperationPending: %t", isPending)

	isDone, err := inspector.IsOperationDone(contractAddress, opID)
	if err != nil {
		log.Fatalf("failed to get operation status: %v", err)
	}
	fmt.Printf("IsOperationDone: %t", isDone)

}

```