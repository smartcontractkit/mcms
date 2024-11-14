package evm

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type Executor struct {
	*Encoder
	*Inspector
	auth *bind.TransactOpts
}

func NewExecutor(encoder *Encoder, client ContractDeployBackend, auth *bind.TransactOpts) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		auth:      auth,
	}
}

func (e *Executor) ExecuteOperation(
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (string, error) {
	if e.Encoder == nil {
		return "", errors.New("Executor was created without an encoder")
	}

	bindOp, err := e.ToGethOperation(nonce, metadata, op)
	if err != nil {
		return "", err
	}

	mcmsC, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	tx, err := mcmsC.Execute(
		e.auth,
		bindOp,
		transformHashes(proof),
	)

	return tx.Hash().Hex(), err
}

func (e *Executor) SetRoot(
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (string, error) {
	if e.Encoder == nil {
		return "", errors.New("Executor was created without an encoder")
	}

	bindMeta, err := e.ToGethRootMetadata(metadata)
	if err != nil {
		return "", err
	}

	mcmsC, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	tx, err := mcmsC.SetRoot(
		e.auth,
		root,
		validUntil,
		bindMeta,
		transformHashes(proof),
		transformSignatures(sortedSignatures),
	)

	return tx.Hash().Hex(), err
}
