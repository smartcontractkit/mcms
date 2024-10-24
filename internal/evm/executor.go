package evm

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms/internal/core"
	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/evm/bindings"
)

type EVMExecutor struct {
	EVMEncoder
	EVMInspector
	auth *bind.TransactOpts
}

func (e *EVMExecutor) ExecuteOperation(
	metadata mcms.ChainMetadata,
	nonce uint32,
	proof []common.Hash,
	operation mcms.ChainOperation,
) (string, error) {
	mcms, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	op, err := e.ToGethOperation(nonce, metadata, operation)
	if err != nil {
		return "", err
	}

	tx, err := mcms.Execute(
		e.auth,
		op,
		core.TransformHashes(proof),
	)

	return tx.Hash().Hex(), err
}

func (e *EVMExecutor) SetRoot(
	metadata mcms.ChainMetadata,
	proof []common.Hash,
	root [32]byte,
	validUntil uint32,
	sortedSignatures []mcms.Signature,
) (string, error) {
	mcms, err := bindings.NewManyChainMultiSig(common.HexToAddress(metadata.MCMAddress), e.client)
	if err != nil {
		return "", err
	}

	tx, err := mcms.SetRoot(
		e.auth,
		root,
		validUntil,
		e.ToGethRootMetadata(metadata),
		core.TransformHashes(proof),
		transformSignatures(sortedSignatures),
	)

	return tx.Hash().Hex(), err
}
