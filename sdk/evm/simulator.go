package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type Simulator struct {
	*Encoder
	*Inspector
}

func NewSimulator(encoder *Encoder, client ContractDeployBackend) (*Simulator, error) {
	if encoder == nil {
		return nil, errors.New("Simulator was created without an encoder")
	}

	if client == nil {
		return nil, errors.New("Simulator was created without an inspector")
	}

	return &Simulator{
		Encoder:   encoder,
		Inspector: NewInspector(client),
	}, nil
}

func (s *Simulator) SimulateSetRoot(
	ctx context.Context,
	originCaller string, // TODO: do we need this or can we just use a random address?
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) error {
	bindMeta, err := s.ToGethRootMetadata(metadata)
	if err != nil {
		return err
	}

	abi, err := bindings.ManyChainMultiSigMetaData.GetAbi()
	if err != nil {
		return err
	}

	data, err := abi.Pack(
		"setRoot",
		root,
		validUntil,
		bindMeta,
		transformHashes(proof),
		transformSignatures(sortedSignatures),
	)
	if err != nil {
		return err
	}

	mcmAddr := common.HexToAddress(metadata.MCMAddress)
	_, err = s.client.CallContract(ctx, ethereum.CallMsg{
		From:  common.HexToAddress(originCaller),
		To:    &mcmAddr,
		Value: big.NewInt(0),
		Data:  data,
	}, nil)

	return err
}

func (s *Simulator) SimulateOperation(
	ctx context.Context,
	metadata types.ChainMetadata,
	operation types.Operation,
) error {
	if s.Encoder == nil {
		return errors.New("Simulator was created without an encoder")
	}

	if s.Inspector == nil {
		return errors.New("Simulator was created without an inspector")
	}

	// Unmarshal the AdditionalFields from the operation
	var additionalFields AdditionalFields
	if err := json.Unmarshal(operation.Transaction.AdditionalFields, &additionalFields); err != nil {
		return err
	}

	toAddr := common.HexToAddress(operation.Transaction.To)
	_, err := s.client.CallContract(ctx, ethereum.CallMsg{
		From:  common.HexToAddress(metadata.MCMAddress),
		To:    &toAddr,
		Value: additionalFields.Value,
		Data:  operation.Transaction.Data,
	}, nil)

	return err
}

func (s *Simulator) SimulateBatchOperation(
	ctx context.Context,
	timelockAddress string,
	batchOperation types.BatchOperation,
) ([]byte, error) {
	if s.Inspector == nil {
		return nil, errors.New("Simulator was created without an inspector")
	}

	ethClient, ok := s.client.(*ethclient.Client) // FIXME: is there a workaround?
	if !ok {
		return nil, fmt.Errorf("unable to cast client to rpc.Client: %T", s.client)
	}

	calls := []transactionArgs{}
	for _, transaction := range batchOperation.Transactions {
		var additionalFields AdditionalFields
		if err := json.Unmarshal(transaction.AdditionalFields, &additionalFields); err != nil {
			return nil, fmt.Errorf("unable to unmarshal additionalFields: %w", err)
		}
		calls = append(calls, transactionArgs{
			From:  ptrTo(common.HexToAddress(timelockAddress)),
			To:    ptrTo(common.HexToAddress(transaction.To)),
			Value: (*hexutil.Big)(additionalFields.Value),
			Data:  ptrTo(hexutil.Bytes(transaction.Data)),
		})
	}
	simulateV1Arg := simOpts{
		BlockStateCalls:        []simBlock{{Calls: calls}},
		ReturnFullTransactions: false,
		TraceTransfers:         false,
		Validation:             false,
	}

	var resp json.RawMessage
	err := ethClient.Client().CallContext(ctx, &resp, "eth_simulateV1", simulateV1Arg, nil)
	if err != nil {
		return nil, fmt.Errorf("eth_simulateV1 failed: %w", err)
	}

	return resp, nil
}

// func (e *EVMSimulator) SimulateChain(
// 	url string,
// ) error {
// 	anvilContainer, err := client.StartAnvil(
// 		[]string{
// 			"--fork-url", url,
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	anvilClient := client.NewRPCClient(anvilContainer.URL, nil) // TODO: headers
// 	anvilClient.AnvilSetStorageAt()
// 	return nil
// }

func ptrTo[T any](v T) *T {
	return &v
}

// types needed by the "eth_simulateV1" endpoint
type transactionArgs struct {
	From                 *common.Address `json:"from,omitempty"`
	To                   *common.Address `json:"to,omitempty"`
	Gas                  *hexutil.Uint64 `json:"gas,omitempty"`
	GasPrice             *hexutil.Big    `json:"gasPrice,omitempty"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	Value                *hexutil.Big    `json:"value,omitempty"`
	Nonce                *hexutil.Uint64 `json:"nonce,omitempty"`
	Data                 *hexutil.Bytes  `json:"data,omitempty"`
}
type simBlock struct {
	Calls []transactionArgs `json:"calls,omitempty"`
}
type simOpts struct {
	BlockStateCalls        []simBlock `json:"blockStateCalls,omitempty"`
	TraceTransfers         bool       `json:"returnFullTransactionObjects,omitempty"`
	Validation             bool       `json:"traceTransfers,omitempty"`
	ReturnFullTransactions bool       `json:"validation,omitempty"`
}
