package mcms

import (
	"sort"

	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// Executable is a struct that represents a proposal that can be executed. It contains all the
// information required to call SetRoot and Execute on the various chains that the proposal
// targets.
type Executable struct {
	*Signable

	Executors map[types.ChainSelector]sdk.Executor
}

// NewExecutable creates a new Executable from a proposal and a map of executors.
func NewExecutable(
	proposal *MCMSProposal,
	executors map[types.ChainSelector]sdk.Executor,
) (*Executable, error) {
	// Get encoders for the proposal
	encoders, err := proposal.GetEncoders()
	if err != nil {
		return nil, err
	}

	// Executor implements Inspector, so we can create a map of Inspectors from Executors
	inspectors := make(map[types.ChainSelector]sdk.Inspector)
	for key, executor := range executors {
		inspectors[key] = executor
	}

	// Create a signable from the proposal
	signable, err := NewSignable(proposal, encoders, inspectors)
	if err != nil {
		return nil, err
	}

	return &Executable{
		Signable:  signable,
		Executors: executors,
	}, nil
}

func (e *Executable) SetRoot(chainSelector types.ChainSelector) (string, error) {
	metadata := e.ChainMetadata[chainSelector]
	metadataHash, err := e.Encoders[chainSelector].HashMetadata(metadata)
	if err != nil {
		return "", err
	}

	proof, err := e.Tree.GetProof(metadataHash)
	if err != nil {
		return "", err
	}

	hash, err := e.SigningHash()
	if err != nil {
		return "", err
	}

	// Sort signatures by recovered address
	sortedSignatures := e.Signatures
	sort.Slice(sortedSignatures, func(i, j int) bool {
		recoveredSignerA, _ := sortedSignatures[i].Recover(hash)
		recoveredSignerB, _ := sortedSignatures[j].Recover(hash)

		return recoveredSignerA.Cmp(recoveredSignerB) < 0
	})

	return e.Executors[chainSelector].SetRoot(
		metadata,
		proof,
		[32]byte(e.Tree.Root.Bytes()),
		e.ValidUntil,
		sortedSignatures,
	)
}

func (e *Executable) Execute(index int) (string, error) {
	transaction := e.Transactions[index]
	chainSelector := transaction.ChainSelector
	metadata := e.ChainMetadata[chainSelector]

	chainNonce, err := safecast.Uint64ToUint32(e.ChainNonce(index))
	if err != nil {
		return "", err
	}

	operationHash, err := e.Encoders[chainSelector].HashOperation(chainNonce, metadata, transaction) // TODO: nonce
	if err != nil {
		return "", err
	}

	proof, err := e.Tree.GetProof(operationHash)
	if err != nil {
		return "", err
	}

	return e.Executors[chainSelector].ExecuteOperation(
		metadata,
		chainNonce,
		proof,
		transaction,
	)
}
