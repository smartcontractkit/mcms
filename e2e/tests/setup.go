package e2e

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
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
		PrivateKeys []string `toml:"private_keys"`
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
	t.Helper()

	setupOnce.Do(func() {
		in, err := framework.Load[Config](t)
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}

		// Fallback to .env if private_keys is not defined in the config
		if len(in.Settings.PrivateKeys) == 0 {
			t.Log("No private_keys found in config. Falling back to .env variable...")
			err = godotenv.Load("../custom_configs/.env")
			if err != nil {
				t.Logf("Failed to load .env file: %v", err)
			}

			envKeys := os.Getenv("PRIVATE_KEYS_E2E")
			if envKeys == "" {
				t.Fatalf("No private_keys found in config,.env or env variables")
			}

			in.Settings.PrivateKeys = strings.Split(envKeys, ",")
			t.Logf("Loaded %d private keys from .env", len(in.Settings.PrivateKeys))
		} else {
			t.Logf("Loaded %d private keys from config", len(in.Settings.PrivateKeys))
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
