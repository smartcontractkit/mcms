package mcms

import (
	"github.com/smartcontractkit/mcms/internal/core/timelock"
	"github.com/smartcontractkit/mcms/types"
)

// Global entry point for any chain timelock proposal. TODO: Complete input
func NewTimelockProposal(chainIdentifier types.ChainIdentifier) (*timelock.TimelockProposal, error) {
	// TODO: map to the correct chain encoder
	return nil, nil
}
