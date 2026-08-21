package stellar

import (
	"context"
	"fmt"
	"time"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	stellarrpc "github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/stellar/go-stellar-sdk/network"
	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
)

const defaultTransactionWindow = 120 * time.Second

// NewInvoker creates the contract invoker used by the Stellar MCMS SDK.
// It follows the same client-plus-signer pattern used by the other MCMS
// families. The deployment framework supplies the RPC client and signer.
func NewInvoker(client *stellarrpc.Client, signer bindings.Signer, selector uint64) (bindings.Invoker, error) {
	chain, exists := chainselectors.StellarChainBySelector(selector)
	if !exists {
		return nil, fmt.Errorf("stellar chain with selector %d does not exist", selector)
	}

	return NewInvokerWithNetworkPassphrase(client, signer, chain.Passphrase)
}

// NewInvokerWithNetworkPassphrase creates an invoker with an explicit network
// passphrase. Deployment frameworks must use this constructor when they hold
// a chain with a custom or local network passphrase.
func NewInvokerWithNetworkPassphrase(client *stellarrpc.Client, signer bindings.Signer, networkPassphrase string) (bindings.Invoker, error) {
	if client == nil {
		return nil, fmt.Errorf("stellar RPC client is nil")
	}
	if signer == nil {
		return nil, fmt.Errorf("stellar signer is nil")
	}
	if networkPassphrase == "" {
		return nil, fmt.Errorf("stellar network passphrase is empty")
	}

	return &rpcInvoker{client: client, signer: signer, networkPassphrase: networkPassphrase}, nil
}

type rpcInvoker struct {
	client            *stellarrpc.Client
	signer            bindings.Signer
	networkPassphrase string
}

var _ bindings.Invoker = (*rpcInvoker)(nil)

func (i *rpcInvoker) InvokeContract(ctx context.Context, contractID, functionName string, args []xdr.ScVal) (*xdr.ScVal, error) {
	op, err := i.invokeOperation(contractID, functionName, args)
	if err != nil {
		return nil, err
	}
	meta, err := i.submit(ctx, op)
	if err != nil {
		return nil, err
	}

	return returnValue(meta)
}

func (i *rpcInvoker) SimulateContract(ctx context.Context, contractID, functionName string, args []xdr.ScVal) (*xdr.ScVal, error) {
	op, err := i.invokeOperation(contractID, functionName, args)
	if err != nil {
		return nil, err
	}
	account, err := i.sourceAccount(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := i.newTransaction(account, op, txnbuild.MinBaseFee)
	if err != nil {
		return nil, err
	}
	xdrTx, err := tx.Base64()
	if err != nil {
		return nil, fmt.Errorf("encode simulation transaction: %w", err)
	}
	result, err := i.client.SimulateTransaction(ctx, protocolrpc.SimulateTransactionRequest{Transaction: xdrTx})
	if err != nil {
		return nil, fmt.Errorf("simulate Stellar contract call: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("simulate Stellar contract call: %s", result.Error)
	}
	if result.RestorePreamble != nil {
		return nil, fmt.Errorf("simulate Stellar contract call requires ledger restore")
	}
	if len(result.Results) == 0 || result.Results[0].ReturnValueXDR == nil || *result.Results[0].ReturnValueXDR == "" {
		return nil, errStellarVoidReturn
	}
	var value xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(*result.Results[0].ReturnValueXDR, &value); err != nil {
		return nil, fmt.Errorf("decode simulation return value: %w", err)
	}

	return &value, nil
}

func (i *rpcInvoker) GetEvents(ctx context.Context, contractID string, startLedger uint32, topics []string) ([]protocolrpc.EventInfo, error) {
	filterTopics := make(protocolrpc.TopicFilter, 0, len(topics)+1)
	for _, topic := range topics {
		filterTopics = append(filterTopics, protocolrpc.SegmentFilter{ScVal: scval.SymbolToScValPtr(topic)})
	}
	wildcard := protocolrpc.WildCardZeroOrMore
	filterTopics = append(filterTopics, protocolrpc.SegmentFilter{Wildcard: &wildcard})
	response, err := i.client.GetEvents(ctx, protocolrpc.GetEventsRequest{
		StartLedger: startLedger,
		Filters: []protocolrpc.EventFilter{{
			EventType:   protocolrpc.EventTypeSet{protocolrpc.EventTypeContract: nil},
			ContractIDs: []string{contractID},
			Topics:      []protocolrpc.TopicFilter{filterTopics},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("get Stellar events: %w", err)
	}

	return response.Events, nil
}

func (i *rpcInvoker) invokeOperation(contractID, functionName string, args []xdr.ScVal) (*txnbuild.InvokeHostFunction, error) {
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return nil, fmt.Errorf("decode Stellar contract ID: %w", err)
	}
	contractAddress := scval.BuildContractScAddress(contractBytes)
	if contractAddress == nil {
		return nil, fmt.Errorf("build Stellar contract address")
	}

	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: *contractAddress,
				FunctionName:    xdr.ScSymbol(functionName),
				Args:            args,
			},
		},
		SourceAccount: i.signer.Address(),
	}, nil
}

func (i *rpcInvoker) sourceAccount(ctx context.Context) (*txnbuild.SimpleAccount, error) {
	key := xdr.LedgerKey{Type: xdr.LedgerEntryTypeAccount, Account: &xdr.LedgerKeyAccount{AccountId: xdr.MustAddress(i.signer.Address())}}
	keyXDR, err := key.MarshalBinaryBase64()
	if err != nil {
		return nil, fmt.Errorf("encode Stellar account key: %w", err)
	}
	response, err := i.client.GetLedgerEntries(ctx, protocolrpc.GetLedgerEntriesRequest{Keys: []string{keyXDR}})
	if err != nil {
		return nil, fmt.Errorf("get Stellar account sequence: %w", err)
	}
	var sequence int64
	if len(response.Entries) > 0 {
		var entry xdr.LedgerEntryData
		entryXDR := response.Entries[0].DataXDR
		if err := xdr.SafeUnmarshalBase64(entryXDR, &entry); err != nil {
			return nil, fmt.Errorf("decode Stellar account entry: %w", err)
		}
		sequence = int64(entry.MustAccount().SeqNum)
	}

	return &txnbuild.SimpleAccount{AccountID: i.signer.Address(), Sequence: sequence}, nil
}

func (i *rpcInvoker) newTransaction(account *txnbuild.SimpleAccount, op txnbuild.Operation, fee int64) (*txnbuild.Transaction, error) {
	return txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        account,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{op},
		BaseFee:              fee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, time.Now().Add(defaultTransactionWindow).Unix())},
	})
}

func (i *rpcInvoker) submit(ctx context.Context, op *txnbuild.InvokeHostFunction) (*xdr.TransactionMeta, error) {
	account, err := i.sourceAccount(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := i.newTransaction(account, op, txnbuild.MinBaseFee)
	if err != nil {
		return nil, fmt.Errorf("build Stellar transaction: %w", err)
	}
	txXDR, err := tx.Base64()
	if err != nil {
		return nil, fmt.Errorf("encode Stellar transaction: %w", err)
	}
	simulation, err := i.client.SimulateTransaction(ctx, protocolrpc.SimulateTransactionRequest{Transaction: txXDR})
	if err != nil {
		return nil, fmt.Errorf("simulate Stellar transaction: %w", err)
	}
	if simulation.Error != "" {
		return nil, fmt.Errorf("simulate Stellar transaction: %s", simulation.Error)
	}
	if simulation.RestorePreamble != nil {
		return nil, fmt.Errorf("submit Stellar transaction requires ledger restore")
	}
	if simulation.TransactionDataXDR != "" {
		var sorobanData xdr.SorobanTransactionData
		if derr := xdr.SafeUnmarshalBase64(simulation.TransactionDataXDR, &sorobanData); derr != nil {
			return nil, fmt.Errorf("decode Soroban transaction data: %w", derr)
		}
		op.Ext = xdr.TransactionExt{V: 1, SorobanData: &sorobanData}
		if len(simulation.Results) > 0 && simulation.Results[0].AuthXDR != nil {
			op.Auth = make([]xdr.SorobanAuthorizationEntry, len(*simulation.Results[0].AuthXDR))
			for idx, authXDR := range *simulation.Results[0].AuthXDR {
				if aerr := xdr.SafeUnmarshalBase64(authXDR, &op.Auth[idx]); aerr != nil {
					return nil, fmt.Errorf("decode Soroban authorization: %w", aerr)
				}
			}
		}
	}
	fee := simulation.MinResourceFee
	if fee < txnbuild.MinBaseFee {
		fee = txnbuild.MinBaseFee
	}
	account, err = i.sourceAccount(ctx)
	if err != nil {
		return nil, err
	}
	tx, err = i.newTransaction(account, op, fee)
	if err != nil {
		return nil, fmt.Errorf("assemble Stellar transaction: %w", err)
	}
	hash, err := network.HashTransactionInEnvelope(tx.ToXDR(), i.networkPassphrase)
	if err != nil {
		return nil, fmt.Errorf("hash Stellar transaction: %w", err)
	}
	signature, err := i.signer.SignDecorated(hash[:])
	if err != nil {
		return nil, fmt.Errorf("sign Stellar transaction: %w", err)
	}
	tx, err = tx.AddSignatureDecorated(signature)
	if err != nil {
		return nil, fmt.Errorf("attach Stellar signature: %w", err)
	}
	signedXDR, err := tx.Base64()
	if err != nil {
		return nil, fmt.Errorf("encode signed Stellar transaction: %w", err)
	}
	submitted, err := i.client.SendTransaction(ctx, protocolrpc.SendTransactionRequest{Transaction: signedXDR})
	if err != nil {
		return nil, fmt.Errorf("submit Stellar transaction: %w", err)
	}
	if submitted.Status != "PENDING" && submitted.Status != "DUPLICATE" {
		return nil, fmt.Errorf("submit Stellar transaction: status %s", submitted.Status)
	}
	deadline := time.Now().Add(defaultTransactionWindow)
	for time.Now().Before(deadline) {
		result, err := i.client.GetTransaction(ctx, protocolrpc.GetTransactionRequest{Hash: submitted.Hash})
		if err == nil {
			switch result.Status {
			case "SUCCESS":
				var meta xdr.TransactionMeta
				if err := xdr.SafeUnmarshalBase64(result.ResultMetaXDR, &meta); err != nil {
					return nil, fmt.Errorf("decode Stellar transaction result: %w", err)
				}

				return &meta, nil
			case "FAILED":
				return nil, fmt.Errorf("stellar transaction failed: %s", submitted.Hash)
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return nil, fmt.Errorf("stellar transaction timed out: %s", submitted.Hash)
}

var errStellarVoidReturn = fmt.Errorf("stellar void return")

func returnValue(meta *xdr.TransactionMeta) (*xdr.ScVal, error) {
	if meta == nil {
		return nil, errStellarVoidReturn
	}
	switch meta.V {
	case 4: //nolint:mnd // protocol version
		if meta.MustV4().SorobanMeta == nil {
			return nil, errStellarVoidReturn
		}

		return meta.MustV4().SorobanMeta.ReturnValue, nil
	case 3: //nolint:mnd // protocol version
		if meta.MustV3().SorobanMeta == nil {
			return nil, errStellarVoidReturn
		}

		return &meta.MustV3().SorobanMeta.ReturnValue, nil
	default:

		return nil, fmt.Errorf("unsupported Stellar transaction meta version: %d", meta.V)
	}
}
