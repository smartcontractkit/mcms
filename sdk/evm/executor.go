package evm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

// Executor is an Executor implementation for EVM chains, allowing for the execution of operations on the MCMS contract
type Executor struct {
	*Encoder
	*Inspector
	auth *bind.TransactOpts
}

// NewExecutor creates a new Executor for EVM chains
func NewExecutor(encoder *Encoder, client ContractDeployBackend, auth *bind.TransactOpts) *Executor {
	return &Executor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		auth:      auth,
	}
}

func (e *Executor) ExecuteOperation(
	ctx context.Context,
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

	opts := *e.auth
	opts.Context = ctx

	tx, err := mcmsC.Execute(&opts, bindOp, transformHashes(proof))
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), err
}

func (e *Executor) SetRoot(
	ctx context.Context,
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

	opts := *e.auth
	opts.Context = ctx

	tx, err := mcmsC.SetRoot(
		&opts,
		root,
		validUntil,
		bindMeta,
		transformHashes(proof),
		transformSignatures(sortedSignatures),
	)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), err
}
