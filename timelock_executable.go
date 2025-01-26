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

	return &TimelockExecutable{
		proposal:  proposal,
		executors: executors,
	}, nil
}

// IsReady checks if ALL the operations in the proposal are ready
// for execution.
// Note: there is some edge cases here where some operations are ready
// but others are not. This is not handled here. Regardless, execution
// should not begin until all operations are ready.
func (t *TimelockExecutable) IsReady(ctx context.Context) error {
	err := t.setPredecessors(ctx)
	if err != nil {
		return fmt.Errorf("unable to set predecessors: %w", err)
	}

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

// Execute executes the operation at the given index.
// Includes an optional callProxyAddress to execute the calls through a proxy.
// If the callProxyAddress is empty string, the calls will be executed directly
// to the timelock.
func (t *TimelockExecutable) Execute(ctx context.Context, index int, callProxyAddress string) (string, error) {
	err := t.setPredecessors(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to set predecessors: %w", err)
	}

	op := t.proposal.Operations[index]

	return t.executors[op.ChainSelector].Execute(
		ctx,
		op,
		t.proposal.TimelockAddresses[op.ChainSelector],
		t.predecessors[index],
		t.proposal.Salt(),
	)
}

func (t *TimelockExecutable) setPredecessors(ctx context.Context) error {
	if len(t.predecessors) == 0 && len(t.executors) > 0 {
		var err error
		var converters = make(map[types.ChainSelector]sdk.TimelockConverter)
		for chainSelector, executor := range t.executors {
			converters[chainSelector], err = newTimelockConverterFromExecutor(chainSelector, executor)
			if err != nil {
				return fmt.Errorf("unable to create converter from executor: %w", err)
			}
		}

		_, t.predecessors, err = t.proposal.Convert(ctx, converters)
		if err != nil {
			return err
		}
	}

	return nil
}
