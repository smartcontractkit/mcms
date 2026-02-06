package chainwrappers

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/xssnick/tonutils-go/ton"

	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	suisuisdk "github.com/smartcontractkit/mcms/sdk/sui"
)

type ChainAccessor interface {
	Selectors() []uint64
	EVMClient(selector uint64) (evmsdk.ContractDeployBackend, bool)
	SolanaClient(selector uint64) (*solrpc.Client, bool)
	AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool)
	SuiClient(selector uint64) (sui.ISuiAPI, suisuisdk.SuiSigner, bool)
	TonClient(selector uint64) (*ton.APIClient, bool)
}
