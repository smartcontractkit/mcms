package internal

import (
	"sort"

	"github.com/smartcontractkit/mcms/internal/core/proposal/mcms"
)

type Executable struct {
	*Signable
	Executors map[mcms.ChainSelector]mcms.Executor
}

func (e *Executable) SetRoot(chainSelector mcms.ChainSelector) (string, error) {
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

	operationHash, err := e.Encoders[chainSelector].HashOperation(0, metadata, transaction) // TODO: nonce
	if err != nil {
		return "", err
	}

	proof, err := e.Tree.GetProof(operationHash)
	if err != nil {
		return "", err
	}

	return e.Executors[chainSelector].ExecuteOperation(
		0, // TODO: nonce
		proof,
		transaction,
	)
}
