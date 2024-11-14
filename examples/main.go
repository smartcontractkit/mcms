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
