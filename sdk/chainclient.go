package sdk

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	solrpc "github.com/gagliardetto/solana-go/rpc"
)

// TODO: this interface should come from chainlink-sui when available
type SuiSigner interface {
	// Sign signs the given message and returns the serialized signature.
	Sign(message []byte) ([]string, error)

	// GetAddress returns the Sui address derived from the signer's public key
	GetAddress() (string, error)
}

type SuiChainClient interface {
	GetClient() sui.ISuiAPI
	GetSigner() SuiSigner
}

type SolanaChainClient interface {
	GetClient() *solrpc.Client
}

type ContractDeployBackend interface {
	bind.ContractBackend
	bind.DeployBackend
}

type EVMChainClient interface {
	GetClient() ContractDeployBackend
}

type ChainClient interface {
	GetClient() aptoslib.AptosRpcClient
}

type BlockChains interface {
	EVMChains() map[uint64]EVMChainClient
	SolanaChains() map[uint64]SolanaChainClient
	AptosChains() map[uint64]ChainClient
	SuiChains() map[uint64]SuiChainClient
}
