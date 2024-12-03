package evm

import (
	"encoding/json"
	"errors"
	"math/big"

	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type Simulator struct {
	*Encoder
	*Inspector
}

func NewSimulator(encoder *Encoder, client ContractDeployBackend) *Simulator {
	return &Simulator{
		Encoder:   encoder,
		Inspector: NewInspector(client),
	}
}

func (s *Simulator) SimulateSetRoot(
	originCaller string, // TODO: do we need this or can we just use a random address?
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) error {
	if s.Encoder == nil {
		return errors.New("Simulator was created without an encoder")
	}

	if s.Inspector == nil {
		return errors.New("Simulator was created without an inspector")
	}

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
	_, err = s.client.CallContract(context.Background(), ethereum.CallMsg{
		From:  common.HexToAddress(originCaller),
		To:    &mcmAddr,
		Value: big.NewInt(0),
		Data:  data,
	}, nil)

	return err
}

func (s *Simulator) SimulateOperation(
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
	_, err := s.client.CallContract(context.Background(), ethereum.CallMsg{
		From:  common.HexToAddress(metadata.MCMAddress),
		To:    &toAddr,
		Value: additionalFields.Value,
		Data:  operation.Transaction.Data,
	}, nil)

	return err
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
