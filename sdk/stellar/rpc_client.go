package stellar

import (
	"context"

	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
)

type RpcClient interface {
	SimulateTransaction(ctx context.Context, req protocolrpc.SimulateTransactionRequest) (protocolrpc.SimulateTransactionResponse, error)
	SendTransaction(ctx context.Context, req protocolrpc.SendTransactionRequest) (protocolrpc.SendTransactionResponse, error)
	GetTransaction(ctx context.Context, req protocolrpc.GetTransactionRequest) (protocolrpc.GetTransactionResponse, error)
	GetLedgerEntries(ctx context.Context, req protocolrpc.GetLedgerEntriesRequest) (protocolrpc.GetLedgerEntriesResponse, error)
	GetEvents(ctx context.Context, req protocolrpc.GetEventsRequest) (protocolrpc.GetEventsResponse, error)
}
