package chainwrappers

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	sol "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/xssnick/tonutils-go/ton"
	tonwallet "github.com/xssnick/tonutils-go/ton/wallet"

	cslclient "github.com/smartcontractkit/chainlink-sui/relayer/client"

	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
)

type ChainAccessor interface {
	Selectors() []uint64
	EVMClient(selector uint64) (evmsdk.ContractDeployBackend, bool)
	EVMSigner(selector uint64) (*evmsdk.TransactOpts, bool)
	SolanaClient(selector uint64) (*solrpc.Client, bool)
	SolanaSigner(selector uint64) (*sol.PrivateKey, bool)
	AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool)
	AptosSigner(selector uint64) (aptoslib.TransactionSigner, bool)
	SuiClient(selector uint64) (cslclient.BindingsClient, bool)
	SuiSigner(selector uint64) (suisdk.SuiSigner, bool)
	TonClient(selector uint64) (ton.APIClientWrapped, bool)
	TonSigner(selector uint64) (*tonwallet.Wallet, bool)
	CantonChain(selector uint64) (cantonsdk.Chain, bool)
}
