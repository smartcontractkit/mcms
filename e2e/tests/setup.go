package e2e

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/stretchr/testify/require"

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
	AptosChain  *blockchain.Input `toml:"aptos_config"`
	SuiChain    *blockchain.Input `toml:"sui_config"`

	Settings struct {
		PrivateKeys []string `toml:"private_keys"`
	} `toml:"settings"`
}

// TestSetup holds common setup for E2E test suites
type TestSetup struct {
	ClientA          *ethclient.Client
	ClientB          *ethclient.Client
	SolanaClient     *rpc.Client
	SolanaWSClient   *ws.Client
	AptosRPCClient   *aptos.NodeClient
	SolanaBlockchain *blockchain.Output
	AptosBlockchain  *blockchain.Output
	SuiClient        sui.ISuiAPI
	SuiBlockchain    *blockchain.Output
	Config
}

// InitializeSharedTestSetup ensures the TestSetup is initialized only once
func InitializeSharedTestSetup(t *testing.T) *TestSetup {
	t.Helper()

	var ethClientA *ethclient.Client
	var ethClientB *ethclient.Client
	// var ethBlockChainOutputA *blockchain.Output
	// var ethBlockChainOutputB *blockchain.Output
	setupOnce.Do(func() {
		// ctx := context.Background()
		in, err := framework.Load[Config](t)
		if err != nil {
			t.Fatalf("Failed to load configuration: %v", err)
		}

		// if in.BlockchainA != nil {
		// 	// Fallback to .env if private_keys is not defined in the config
		// 	if len(in.Settings.PrivateKeys) == 0 {
		// 		t.Log("No private_keys found in config. Falling back to .env variable...")
		// 		err = godotenv.Load("../custom_configs/.env")
		// 		if err != nil {
		// 			t.Logf("Failed to load .env file: %v", err)
		// 		}

		// 		envKeys := os.Getenv("PRIVATE_KEYS_E2E")
		// 		if envKeys == "" {
		// 			t.Fatalf("No private_keys found in config,.env or env variables")
		// 		}

		// 		in.Settings.PrivateKeys = strings.Split(envKeys, ",")
		// 		t.Logf("Loaded %d private keys from .env", len(in.Settings.PrivateKeys))
		// 	} else {
		// 		t.Logf("Loaded %d private keys from config", len(in.Settings.PrivateKeys))
		// 	}

		// 	// Initialize the blockchain A
		// 	ethBlockChainOutputA, err = blockchain.NewBlockchainNetwork(in.BlockchainA)
		// 	if err != nil {
		// 		t.Fatalf("Failed to initialize blockchain network: %v", err)
		// 	}

		// 	// Initialize the blockchain B
		// 	ethBlockChainOutputB, err = blockchain.NewBlockchainNetwork(in.BlockchainB)
		// 	if err != nil {
		// 		t.Fatalf("Failed to initialize blockchain network: %v", err)
		// 	}

		// 	// Initialize Ethereum client A
		// 	wsURLA := ethBlockChainOutputA.Nodes[0].HostWSUrl
		// 	ethClientA, err = ethclient.DialContext(context.Background(), wsURLA)
		// 	if err != nil {
		// 		t.Fatalf("Failed to initialize Ethereum client: %v", err)
		// 	}

		// 	// Initialize Ethereum client B
		// 	wsURLB := ethBlockChainOutputB.Nodes[0].HostWSUrl
		// 	ethClientB, err = ethclient.DialContext(context.Background(), wsURLB)
		// 	if err != nil {
		// 		t.Fatalf("Failed to initialize Ethereum client: %v", err)
		// 	}
		// }

		// var solanaClient *rpc.Client
		// var solanaWsClient *ws.Client
		// var solanaBlockChainOutput *blockchain.Output
		// if in.SolanaChain != nil {
		// 	if in.SolanaChain.ContractsDir == "" {
		// 		in.SolanaChain.ContractsDir = filepath.Join(ProjectRoot, "/e2e/artifacts/solana")
		// 	}

		// 	// Initialize Solana client
		// 	solanaBlockChainOutput, err = blockchain.NewBlockchainNetwork(in.SolanaChain)
		// 	if err != nil {
		// 		t.Fatalf("Failed to initialize solana blockchain: %v", err)
		// 	}

		// 	solanaClient = rpc.New(solanaBlockChainOutput.Nodes[0].HostHTTPUrl)
		// 	solanaWsClient, err = ws.Connect(ctx, solanaBlockChainOutput.Nodes[0].HostWSUrl)
		// 	if err != nil {
		// 		t.Fatalf("Failed to initialize Solana WS client: %v", err)
		// 	}

		// 	// Test the connection by checking the health of the RPC node
		// 	var health string
		// 	health, err = solanaClient.GetHealth(ctx)
		// 	if err != nil {
		// 		t.Fatalf("Failed to connect to Solana RPC: %v", err)
		// 	}

		// 	if health == rpc.HealthOk {
		// 		t.Log("Connection to Solana RPC is successful!")
		// 	} else {
		// 		t.Fatal("Connection established, but node health is not OK.")
		// 	}
		// }

		// var (
		// 	aptosClient           *aptos.NodeClient
		// 	aptosBlockchainOutput *blockchain.Output
		// )
		// if in.AptosChain != nil {
		// 	aptosBlockchainOutput, err = blockchain.NewBlockchainNetwork(in.AptosChain)
		// 	require.NoError(t, err, "Failed to initialize Aptos blockchain")

		// 	nodeUrl := fmt.Sprintf("%v/v1", aptosBlockchainOutput.Nodes[0].HostHTTPUrl)

		// 	aptosClient, err = aptos.NewNodeClient(nodeUrl, 0)
		// 	require.NoError(t, err, "Failed to initialize Aptos RPC client")

		// 	// Test liveness, will also fetch ChainID
		// 	t.Logf("Initialized Aptos RPC client @ %s", nodeUrl)
		// 	info, err := aptosClient.Info()
		// 	require.NoError(t, err, "Failed to get Aptos node info")
		// 	require.NotEmpty(t, info.LedgerVersionStr)
		// 	in.AptosChain.ChainID = strconv.FormatUint(uint64(info.ChainId), 10)
		// }

		var (
			suiClient           sui.ISuiAPI
			suiBlockchainOutput *blockchain.Output
		)
		if in.SuiChain != nil {
			suiBlockchainOutput, err = blockchain.NewBlockchainNetwork(in.SuiChain)
			require.NoError(t, err, "Failed to initialize Sui blockchain")

			nodeUrl := suiBlockchainOutput.Nodes[0].HostHTTPUrl
			suiClient = sui.NewSuiClient(nodeUrl)

			// Test liveness, will also fetch ChainID
			t.Logf("Initialized Sui RPC client @ %s", nodeUrl)
		}

		sharedSetup = &TestSetup{
			ClientA: ethClientA,
			ClientB: ethClientB,
			// SolanaClient:     solanaClient,
			// SolanaWSClient:   solanaWsClient,
			// AptosRPCClient:   aptosClient,
			// SolanaBlockchain: solanaBlockChainOutput,
			// AptosBlockchain:  aptosBlockchainOutput,
			SuiClient:     suiClient,
			SuiBlockchain: suiBlockchainOutput,
			Config:        *in,
		}
	})

	return sharedSetup
}
