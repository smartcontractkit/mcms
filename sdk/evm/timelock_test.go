package evm

import "testing"

func Test_TimelockProposal(t *testing.T) {

}

// import (
// 	"testing"

// 	"github.com/ethereum/go-ethereum/ethclient/simulated"
// )

// // TODO: Create this generic function
// func setupSimulatedBackend() *simulated.Backend {
// 	return simulated.NewBackend()
// }

// func Test_TimelockProposal(t *testing.T) {
// 	// Example on how to use Timelock with MCMS

// 	// Create a new TimelockProposal
// 	timelockProposal, err := NewTimelockProposal(Schedule, []BatchChainOperation{}, "1d")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a new EVM Timelock Proposal
// 	evmTimelockProposal := NewEVMTimelockProposal(*timelockProposal)
// 	txData := evmTimelockProposal.Encode()

// }
