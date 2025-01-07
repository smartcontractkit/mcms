package mcms

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// TimelockExecutable is a struct that represents a proposal that can be executed
// with a timelock. It contains all the information required to call executeBatch
// for scheduled calls
type TimelockExecutable struct {
	proposal     *TimelockProposal
	predecessors []common.Hash
	executors    map[types.ChainSelector]sdk.TimelockExecutor
}

// NewTimelockExecutable creates a new TimelockExecutable from a proposal and a map of executors.
func NewTimelockExecutable(
	proposal *TimelockProposal,
	executors map[types.ChainSelector]sdk.TimelockExecutor,
) (*TimelockExecutable, error) {
	if proposal.Action != types.TimelockActionSchedule {
		return nil, fmt.Errorf("TimelockExecutable can only be created from a TimelockProposal with action 'schedule'")
	}

	_, predecessors, err := proposal.Convert()
	if err != nil {
		return nil, err
	}

	return &TimelockExecutable{
		proposal:     proposal,
		executors:    executors,
		predecessors: predecessors,
	}, nil
}

// IsReady checks if ALL the operations in the proposal are ready
// for execution.
// Note: there is some edge cases here where some operations are ready
// but others are not. This is not handled here. Regardless, execution
// should not begin until all operations are ready.
func (t *TimelockExecutable) IsReady(ctx context.Context) error {
	for i, op := range t.proposal.Operations {
		cs := op.ChainSelector
		timelock := t.proposal.TimelockAddresses[cs]
		isOpReady, err := t.executors[cs].IsOperationReady(ctx, timelock, t.predecessors[i+1])
		if err != nil {
			return err
		}

		if !isOpReady {
			return fmt.Errorf("operation %d is not ready", i)
		}
	}

	return nil
}

func (t *TimelockExecutable) Execute(ctx context.Context, index int) (string, error) {
	op := t.proposal.Operations[index]
	return t.executors[op.ChainSelector].Execute(
		ctx,
		op,
		t.proposal.TimelockAddresses[op.ChainSelector],
		t.predecessors[index],
		t.proposal.Salt(),
	)
}
