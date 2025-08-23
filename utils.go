package mcms

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/mcms/types"
)

// Applies the EIP191 prefix to the payload and hashes it.
func toEthSignedMessageHash(payload []byte) common.Hash {
	// Add the Ethereum signed message prefix
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	data := append(prefix, payload...)

	// Hash the prefixed message
	return crypto.Keccak256Hash(data)
}

func generateQueuedProposalStartingOpCounts[T ProposalInterface](predecessorProposals []T) map[types.ChainSelector]uint64 {
	// Set the transaction counts for each chain selector
	startingOpCounts := make(map[types.ChainSelector]uint64)
	for _, pred := range predecessorProposals {
		chainMetadata := pred.ChainMetadatas()
		for chainSelector, count := range pred.TransactionCounts() {
			if _, ok := startingOpCounts[chainSelector]; !ok {
				startingOpCounts[chainSelector] = chainMetadata[chainSelector].StartingOpCount
			}

			startingOpCounts[chainSelector] += count
		}
	}

	return startingOpCounts
}

func decodeAndValidateProposal[T ProposalInterface](reader io.Reader) (T, error) {
	// Decode the proposal
	var proposal T
	if err := json.NewDecoder(reader).Decode(&proposal); err != nil {
		return proposal, fmt.Errorf("failed to decode proposal: %w", err)
	}

	// Validate the proposal
	if err := proposal.Validate(); err != nil {
		return proposal, fmt.Errorf("failed to validate proposal: %w", err)
	}

	return proposal, nil
}

func newProposal[T ProposalInterface](r io.Reader, predecessors []io.Reader) (T, error) {
	p, err := decodeAndValidateProposal[T](r)
	if err != nil {
		return p, fmt.Errorf("failed to decode and validate target proposal: %w", err)
	}

	predecessorProposals := make([]T, len(predecessors))
	for i, pred := range predecessors {
		predObj, err := decodeAndValidateProposal[T](pred)
		if err != nil {
			return p, fmt.Errorf("failed to decode and validate predecessor proposal %d: %w", i, err)
		}

		predecessorProposals[i] = predObj
	}

	startingOpCounts := generateQueuedProposalStartingOpCounts(predecessorProposals)

	// Set the starting op count for each chain selector in the new proposal
	for chainSelector, chainMetadata := range p.ChainMetadatas() {
		if count, ok := startingOpCounts[chainSelector]; ok {
			chainMetadata.StartingOpCount = count
		}

		p.setChainMetadata(chainSelector, chainMetadata)
	}

	return p, nil
}

func pointerTo[T any](v T) *T {
	return &v
}
