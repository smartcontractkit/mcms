package chainwrappers

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	sol "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	cldfcanton "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/xssnick/tonutils-go/ton"
	tonwallet "github.com/xssnick/tonutils-go/ton/wallet"

	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
)

type ChainAccessor interface {
	Selectors() []uint64
	EVMClient(selector uint64) (evmsdk.ContractDeployBackend, bool)
	EVMSigner(selector uint64) (*bind.TransactOpts, bool)
	SolanaClient(selector uint64) (*solrpc.Client, bool)
	SolanaSigner(selector uint64) (*sol.PrivateKey, bool)
	AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool)
	AptosSigner(selector uint64) (aptoslib.TransactionSigner, bool)
	SuiClient(selector uint64) (sui.ISuiAPI, bool)
	SuiSigner(selector uint64) (suisdk.SuiSigner, bool)
	TonClient(selector uint64) (ton.APIClientWrapped, bool)
	TonSigner(selector uint64) (*tonwallet.Wallet, bool)
	CantonChain(selector uint64) (cldfcanton.Chain, bool)
}
