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
	proposal         *Proposal
	executors        map[types.ChainSelector]sdk.Executor
	encoders         map[types.ChainSelector]sdk.Encoder
	instanceEncoders map[instanceKey]sdk.Encoder
	tree             *merkle.Tree
	txNonces         []uint64
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

	// Generate the per-instance encoders (used for root metadata hashing)
	instanceEncoders, err := proposal.GetInstanceEncoders()
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
		proposal:         proposal,
		executors:        executors,
		encoders:         encoders,
		instanceEncoders: instanceEncoders,
		tree:             tree,
		txNonces:         txNonces,
	}, nil
}

// MCMAddresses returns the MCM instance addresses for a chain selector: the primary MCM
// followed by any additional MCM instances. Callers should call SetRoot once per address.
func (e *Executable) MCMAddresses(chainSelector types.ChainSelector) []string {
	metadata, ok := e.proposal.ChainMetadata[chainSelector]
	if !ok {
		return nil
	}

	instances := metadata.AllMCMs()
	addresses := make([]string, 0, len(instances))
	for _, instance := range instances {
		addresses = append(addresses, instance.MCMAddress)
	}

	return addresses
}

// SetRoot calls SetRoot on the chain's primary MCM instance. For chains with multiple
// MCM instances, use SetRootForMCM to target a specific instance.
func (e *Executable) SetRoot(ctx context.Context, chainSelector types.ChainSelector) (types.TransactionResult, error) {
	return e.SetRootForMCM(ctx, chainSelector, "")
}

// SetRootForMCM calls SetRoot on the MCM instance identified by mcmAddress. An empty
// mcmAddress targets the chain's primary MCM instance.
func (e *Executable) SetRootForMCM(
	ctx context.Context, chainSelector types.ChainSelector, mcmAddress string,
) (types.TransactionResult, error) {
	metadata, ok := e.proposal.ChainMetadata[chainSelector]
	if !ok {
		return types.TransactionResult{}, NewChainMetadataNotFoundError(chainSelector)
	}

	instanceMetadata, ok := metadata.GetMCM(mcmAddress)
	if !ok {
		return types.TransactionResult{}, fmt.Errorf(
			"chain %d: mcmAddress %q does not match the chain's primary MCM or any additional MCM instance",
			chainSelector, mcmAddress)
	}

	metadata = instanceMetadata

	// Use the per-instance encoder so the metadata leaf hashes this instance's own
	// postOpCount (StartingOpCount + instance op count).
	metadataHash, err := e.instanceEncoders[instanceKey{chainSelector: chainSelector, mcmAddress: metadata.MCMAddress}].
		HashMetadata(metadata)
	if err != nil {
		return types.TransactionResult{}, err
	}

	proof, err := e.tree.GetProof(metadataHash)
	if err != nil {
		return types.TransactionResult{}, err
	}

	hash, err := e.proposal.SigningHash() //nolint:contextcheck,nolintlint //OPT-400
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

	root := [32]byte(e.tree.Root.Bytes())
	executor := e.executors[chainSelector]

	// For chains with multiple MCM instances, the on-chain root metadata must carry the
	// instance's own postOpCount (matching the metadata leaf hashed above). Executors
	// derive postOpCount from their chain-wide tx count by default, so they must
	// implement sdk.InstanceExecutor to support multi-instance set-root.
	if len(e.proposal.ChainMetadata[chainSelector].AdditionalMCMs) > 0 {
		instanceExecutor, ok := executor.(sdk.InstanceExecutor)
		if !ok {
			return types.TransactionResult{}, fmt.Errorf(
				"chain %d: executor %T does not support multiple MCM instances (sdk.InstanceExecutor)",
				chainSelector, executor)
		}

		instanceOpCount := e.proposal.TransactionCountsByInstance()[instanceKey{
			chainSelector: chainSelector,
			mcmAddress:    metadata.MCMAddress,
		}]

		return instanceExecutor.SetRootForInstance(
			ctx,
			metadata,
			instanceOpCount,
			proof,
			root,
			e.proposal.ValidUntil,
			sortedSignatures,
		)
	}

	return executor.SetRoot(
		ctx,
		metadata,
		proof,
		root,
		e.proposal.ValidUntil,
		sortedSignatures,
	)
}

func (e *Executable) Execute(ctx context.Context, index int) (types.TransactionResult, error) {
	op := e.proposal.Operations[index]
	chainSelector := op.ChainSelector

	metadata, err := e.proposal.mcmMetadataForOp(op)
	if err != nil {
		return types.TransactionResult{}, err
	}

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
