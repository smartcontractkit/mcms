package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
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
	sharedSetup *TestSetup
	setupOnce   sync.Once
)

// Config defines the blockchain configuration
type Config struct {
	BlockchainA *blockchain.Input `toml:"evm_config" validate:"required"`
	SolanaChain *SolConfig        `toml:"solana_config" validate:"required"`
	Settings    struct {
		PrivateKeys []string `toml:"private_keys"`
	} `toml:"settings"`
}

// TestSetup holds common setup for E2E test suites
type TestSetup struct {
	Client         *ethclient.Client
	ClientSolana   *rpc.Client
	ClientWSSolana *ws.Client
	Blockchain     *blockchain.Output
	Config
}

// InitializeSharedTestSetup ensures the TestSetup is initialized only once
func InitializeSharedTestSetup(t *testing.T) *TestSetup {
	t.Helper()

	setupOnce.Do(func() {
		ctx := context.Background()
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

		// Initialize Solana Chain
		// TODO: this should be moved to the framework, inside  blockchain.NewBlockchainNetwork()
		solanaChain, err := in.SolanaChain.newSolana()
		if err != nil {
			t.Fatalf("Failed to initialize Solana chain: %v", err)
		}
		// Initialize Solana client
		httpURLSol := solanaChain.Nodes[0].HostHTTPUrl
		wsURLSol := solanaChain.Nodes[0].HostWSUrl
		clientSol := rpc.New(httpURLSol)
		clientSolWS, err := ws.Connect(ctx, wsURLSol)
		if err != nil {
			t.Fatalf("Failed to connect to Solana WS: %v", err)
		}
		// Test the connection by checking the health of the RPC node
		health, err := clientSol.GetHealth(ctx)
		if err != nil {
			log.Fatalf("Failed to connect to Solana RPC: %v", err)
		}

		if health == rpc.HealthOk {
			fmt.Println("Connection to Solana RPC is successful!")
		} else {
			fmt.Println("Connection established, but node health is not OK.")
		}

		// Alternatively, get the node version to confirm connectivity
		version, err := clientSol.GetVersion(ctx)
		if err != nil {
			log.Fatalf("Failed to retrieve Solana version: %v", err)
		}
		fmt.Printf("Connected to Solana RPC. Node version: %s\n", version.SolanaCore)
		sharedSetup = &TestSetup{
			Client:         client,
			ClientSolana:   clientSol,
			ClientWSSolana: clientSolWS,
			Blockchain:     bc,
			Config:         *in,
		}
	})

	return sharedSetup
}
