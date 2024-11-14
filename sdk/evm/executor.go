package evm

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type EVMExecutor struct {
	*Encoder
	*Inspector
	auth *bind.TransactOpts
}

func NewEVMExecutor(encoder *Encoder, client ContractDeployBackend, auth *bind.TransactOpts) *EVMExecutor {
	return &EVMExecutor{
		Encoder:   encoder,
		Inspector: NewInspector(client),
		auth:      auth,
	}
}

func (e *EVMExecutor) ExecuteOperation(
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	op types.Operation,
) (string, error) {
	if e.Encoder == nil {
		return "", errors.New("EVMExecutor was created without an encoder")
	}

	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	bindOp, err := e.ToGethOperation(nonce, metadata, op)
	if err != nil {
		return "", err
	}

	tx, err := mcmsObj.Execute(
		e.auth,
		bindOp,
		transformHashes(proof),
	)

	return tx.Hash().Hex(), err
}

func (e *EVMExecutor) SetRoot(
	metadata types.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []types.Signature,
) (string, error) {
	if e.Encoder == nil {
		return "", errors.New("EVMExecutor was created without an encoder")
	}

	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	tx, err := mcmsObj.SetRoot(
		e.auth,
		root,
		validUntil,
		e.ToGethRootMetadata(metadata),
		transformHashes(proof),
		transformSignatures(sortedSignatures),
	)

	return tx.Hash().Hex(), err
}
