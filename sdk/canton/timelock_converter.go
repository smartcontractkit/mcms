package canton

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

// TimelockConverter is a stub implementation for Canton timelock conversion.
// ConvertBatchToChainOperations returns "not implemented" until Phase C.
type TimelockConverter struct{}

// NewTimelockConverter creates a new TimelockConverter (stub).
func NewTimelockConverter() *TimelockConverter {
	return &TimelockConverter{}
}

func (t *TimelockConverter) ConvertBatchToChainOperations(
	ctx context.Context,
	metadata types.ChainMetadata,
	bop types.BatchOperation,
	timelockAddress string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	return nil, common.Hash{}, errors.New("not implemented on Canton")
}
