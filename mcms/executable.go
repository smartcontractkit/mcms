package mcms

import (
	"sort"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
	"github.com/smartcontractkit/mcms/internal/utils/safecast"
	"github.com/smartcontractkit/mcms/types"
)

type Executable struct {
	*Signable
	Executors map[types.ChainSelector]mcms.Executor
}

func NewExecutable(
	signable *Signable,
	executors map[types.ChainSelector]mcms.Executor,
) *Executable {
	return &Executable{
		Signable:  signable,
		Executors: executors,
	}
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
