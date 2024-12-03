package evm

import (
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
	originCaller string, // TODO: do we need this or can we just use a random address?
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	operation types.Operation,
) error {
	if s.Encoder == nil {
		return errors.New("Simulator was created without an encoder")
	}

	if s.Inspector == nil {
		return errors.New("Simulator was created without an inspector")
	}

	bindOp, err := s.ToGethOperation(nonce, metadata, operation)
	if err != nil {
		return err
	}

	abi, err := bindings.ManyChainMultiSigMetaData.GetAbi()
	if err != nil {
		return err
	}

	data, err := abi.Pack(
		"execute",
		bindOp,
		transformHashes(proof),
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
