package e2e

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/joho/godotenv"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
)

// Shared test setup
var (
	sharedSetup       *TestSetup
	setupOnce         sync.Once
	_, fileName, _, _ = runtime.Caller(0)
	ProjectRoot       = filepath.Dir(filepath.Dir(filepath.Dir(fileName)))
)

// Config defines the blockchain configuration
type Config struct {
	BlockchainA *blockchain.Input `toml:"evm_config_a"`
	BlockchainB *blockchain.Input `toml:"evm_config_b"`
	SolanaChain *blockchain.Input `toml:"solana_config"`
	Settings    struct {
		PrivateKeys []string `toml:"private_keys"`
	} `toml:"settings"`
}

// TestSetup holds common setup for E2E test suites
type TestSetup struct {
	ClientA          *ethclient.Client
	ClientB          *ethclient.Client
	SolanaClient     *rpc.Client
	SolanaWSClient   *ws.Client
	SolanaBlockchain *blockchain.Output
	Config
}

// InitializeSharedTestSetup ensures the TestSetup is initialized only once
func InitializeSharedTestSetup(t *testing.T) *TestSetup {
	t.Helper()

	var ethClientA *ethclient.Client
	var ethClientB *ethclient.Client
	var ethBlockChainOutputA *blockchain.Output
	var ethBlockChainOutputB *blockchain.Output
	setupOnce.Do(func() {
		ctx := context.Background()
		in, err := framework.Load[Config](t)
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}

		if in.BlockchainA != nil {
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

			// Initialize the blockchain A
			ethBlockChainOutputA, err = blockchain.NewBlockchainNetwork(in.BlockchainA)
			if err != nil {
				t.Fatalf("Failed to initialize blockchain network: %v", err)
			}

			// Initialize the blockchain B
			ethBlockChainOutputB, err = blockchain.NewBlockchainNetwork(in.BlockchainB)
			if err != nil {
				t.Fatalf("Failed to initialize blockchain network: %v", err)
			}

			// Initialize Ethereum client A
			wsURLA := ethBlockChainOutputA.Nodes[0].HostWSUrl
			ethClientA, err = ethclient.DialContext(context.Background(), wsURLA)
			if err != nil {
				t.Fatalf("Failed to initialize Ethereum client: %v", err)
			}

			// Initialize Ethereum client B
			wsURLB := ethBlockChainOutputB.Nodes[0].HostWSUrl
			ethClientB, err = ethclient.DialContext(context.Background(), wsURLB)
			if err != nil {
				t.Fatalf("Failed to initialize Ethereum client: %v", err)
			}
		}

		var solanaClient *rpc.Client
		var solanaWsClient *ws.Client
		var solanaBlockChainOutput *blockchain.Output
		if in.SolanaChain != nil {
			if in.SolanaChain.ContractsDir == "" {
				in.SolanaChain.ContractsDir = filepath.Join(ProjectRoot, "/e2e/artifacts/solana")
			}

			// Initialize Solana client
			solanaBlockChainOutput, err = blockchain.NewBlockchainNetwork(in.SolanaChain)
			if err != nil {
				t.Fatalf("Failed to initialize solana blockchain: %v", err)
			}

			solanaClient = rpc.New(solanaBlockChainOutput.Nodes[0].HostHTTPUrl)
			solanaWsClient, err = ws.Connect(ctx, solanaBlockChainOutput.Nodes[0].HostWSUrl)
			if err != nil {
				t.Fatalf("Failed to initialize Solana WS client: %v", err)
			}

			// Test the connection by checking the health of the RPC node
			health, err := solanaClient.GetHealth(ctx)
			if err != nil {
				t.Fatalf("Failed to connect to Solana RPC: %v", err)
			}

			if health == rpc.HealthOk {
				t.Log("Connection to Solana RPC is successful!")
			} else {
				t.Fatal("Connection established, but node health is not OK.")
			}
		}

		sharedSetup = &TestSetup{
			ClientA:          ethClientA,
			ClientB:          ethClientB,
			SolanaClient:     solanaClient,
			SolanaWSClient:   solanaWsClient,
			SolanaBlockchain: solanaBlockChainOutput,
			Config:           *in,
		}
	})

	return sharedSetup
}
