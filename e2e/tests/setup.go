//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
)

// Shared test setup
var (
	sharedSetup *TestSetup
	setupOnce   sync.Once
)

// Config defines the blockchain configuration
type Config struct {
	BlockchainA *blockchain.Input `toml:"evm_config" validate:"required"`
	Settings    struct {
		PrivateKeys []string `toml:"private_keys" validate:"required"`
	} `toml:"settings"`
}

// TestSetup holds common setup for E2E test suites
type TestSetup struct {
	Client     *ethclient.Client
	Blockchain *blockchain.Output
	Config
}

// InitializeSharedTestSetup ensures the TestSetup is initialized only once
func InitializeSharedTestSetup(t *testing.T) *TestSetup {
	setupOnce.Do(func() {
		in, err := framework.Load[Config](t)
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}

		// Initialize the blockchain
		bc, err := blockchain.NewBlockchainNetwork(in.BlockchainA)
		if err != nil {
			t.Fatalf("Failed to initialize blockchain network: %v", err)
		}

		// Initialize Ethereum client
		wsURL := bc.Nodes[0].HostWSUrl
		client, err := ethclient.DialContext(context.Background(), wsURL)
		if err != nil {
			t.Fatalf("Failed to initialize Ethereum client: %v", err)
		}

		sharedSetup = &TestSetup{
			Client:     client,
			Blockchain: bc,
			Config:     *in,
		}
	})

	return sharedSetup
}
