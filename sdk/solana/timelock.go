package solana

import "github.com/smartcontractkit/mcms/internal/core/timelock"

type SolanaTimelockProposal struct {
	proposal timelock.TimelockProposal
}

func NewSolanaTimelockProposal(proposal timelock.TimelockProposal) *SolanaTimelockProposal {
	return &SolanaTimelockProposal{
		proposal: proposal,
	}
}

// Ensures SolanaTimelockProposal follows TimelockEncoder interface
var _ timelock.TimelockEncoder = (*SolanaTimelockProposal)(nil)

func (p *SolanaTimelockProposal) Encode() ([]byte, error) {
	// TODO: Implement the Solana specific encoding logic
	return nil, nil
}
