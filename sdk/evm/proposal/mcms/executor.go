package evm

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

type EVMExecutor struct {
	*EVMEncoder
	*EVMInspector
	auth *bind.TransactOpts
}

func NewEVMExecutor(encoder *EVMEncoder, client evm.ContractDeployBackend, auth *bind.TransactOpts) *EVMExecutor {
	return &EVMExecutor{
		EVMEncoder:   encoder,
		EVMInspector: NewEVMInspector(client),
		auth:         auth,
	}
}

func (e *EVMExecutor) ExecuteOperation(
	metadata types.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	operation types.ChainOperation,
) (string, error) {
	if e.EVMEncoder == nil {
		return "", errors.New("EVMExecutor was created without an encoder")
	}

	mcmsObj, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	op, err := e.ToGethOperation(nonce, metadata, operation)
	if err != nil {
		return "", err
	}

	tx, err := mcmsObj.Execute(
		e.auth,
		op,
		core.TransformHashes(proof),
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
	if e.EVMEncoder == nil {
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
		core.TransformHashes(proof),
		evm.TransformSignatures(sortedSignatures),
	)

	return tx.Hash().Hex(), err
}
