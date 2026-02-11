package mcms

import (
	"context"
	"fmt"
	"slices"

	"github.com/smartcontractkit/mcms/internal/core/merkle"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// Executable is a struct that represents a proposal that can be executed. It contains all the
// information required to call SetRoot and Execute on the various chains that the proposal
// targets.
type Executable struct {
	proposal  *Proposal
	executors map[types.ChainSelector]sdk.Executor
	encoders  map[types.ChainSelector]sdk.Encoder
	tree      *merkle.Tree
	txNonces  []uint64
}

// NewExecutable creates a new Executable from a proposal and a map of executors.
func NewExecutable(
	proposal *Proposal,
	executors map[types.ChainSelector]sdk.Executor,
) (*Executable, error) {
	// Generate the encoders from the proposal
	encoders, err := proposal.GetEncoders()
	if err != nil {
		return nil, err
	}

	// Generate the tx nonces from the proposal
	txNonces, err := proposal.TransactionNonces()
	if err != nil {
		return nil, err
	}

	// Generate the tree from the proposal
	tree, err := proposal.MerkleTree()
	if err != nil {
		return nil, err
	}

	return &Executable{
		proposal:  proposal,
		executors: executors,
		encoders:  encoders,
		tree:      tree,
		txNonces:  txNonces,
	}, nil
}

func (e *Executable) SetRoot(ctx context.Context, chainSelector types.ChainSelector) (types.TransactionResult, error) {
	metadata := e.proposal.ChainMetadata[chainSelector]

	metadataHash, err := e.encoders[chainSelector].HashMetadata(metadata)
	if err != nil {
		return types.TransactionResult{}, err
	}

	proof, err := e.tree.GetProof(metadataHash)
	if err != nil {
		return types.TransactionResult{}, err
	}

	// TODO: we should pass the context once we remove background context in the remote chain selectors api
	hash, err := e.proposal.SigningHash() //nolint:contextcheck,nolintlint
	if err != nil {
		return types.TransactionResult{}, err
	}

	// Sort signatures by recovered address
	sortedSignatures := slices.Clone(e.proposal.Signatures) // Clone so we don't modify the original
	slices.SortFunc(sortedSignatures, func(a, b types.Signature) int {
		recoveredSignerA, _ := a.Recover(hash)
		recoveredSignerB, _ := b.Recover(hash)

		return recoveredSignerA.Cmp(recoveredSignerB)
	})

	return e.executors[chainSelector].SetRoot(
		ctx,
		metadata,
		proof,
		[32]byte(e.tree.Root.Bytes()),
		e.proposal.ValidUntil,
		sortedSignatures,
	)
}

func (e *Executable) Execute(ctx context.Context, index int) (types.TransactionResult, error) {
	op := e.proposal.Operations[index]
	chainSelector := op.ChainSelector
	metadata := e.proposal.ChainMetadata[chainSelector]

	txNonce, err := safecast.Uint64ToUint32(e.txNonces[index])
	if err != nil {
		return types.TransactionResult{}, err
	}

	operationHash, err := e.encoders[chainSelector].HashOperation(txNonce, metadata, op)
	if err != nil {
		return types.TransactionResult{}, err
	}

	proof, err := e.tree.GetProof(operationHash)
	if err != nil {
		return types.TransactionResult{}, err
	}

	return e.executors[chainSelector].ExecuteOperation(
		ctx,
		metadata,
		txNonce,
		proof,
		op,
	)
}

func (e *Executable) TxNonce(index int) (uint64, error) {
	if index >= len(e.txNonces) {
		return 0, fmt.Errorf("index out of range: %d >= %d", index, len(e.txNonces))
	}

	return e.txNonces[index], nil
}
