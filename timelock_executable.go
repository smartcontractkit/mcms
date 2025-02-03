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
	predecessors map[types.ChainSelector][]common.Hash
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
		isOpReady, err := t.executors[cs].IsOperationReady(ctx, timelock, t.predecessors[cs][i+1])
		if err != nil {
			return err
		}

		if !isOpReady {
			return fmt.Errorf("operation %d is not ready", i)
		}
	}

	return nil
}

type Option func(*executeOptions)

type executeOptions struct {
	callProxy string
}

func WithCallProxy(address string) Option {
	return func(opts *executeOptions) {
		opts.callProxy = address
	}
}

// GetChainSpecificIndex gets the index of the operation in the context of the given chain.
func (t *TimelockExecutable) GetChainSpecificIndex(index int) int {
	op := t.proposal.Operations[index]
	chainSelector := op.ChainSelector
	chainSpecificIndex := 0
	for i, op := range t.proposal.Operations {
		if op.ChainSelector == chainSelector && i <= index {
			chainSpecificIndex++
		}
	}
	return chainSpecificIndex
}

// Execute executes the operation at the given index.
// Includes an option to set callProxy to execute the calls through a proxy.
// If the callProxy is not set, the calls will be executed directly
// to the timelock.
func (t *TimelockExecutable) Execute(ctx context.Context, index int, opts ...Option) (types.TransactionResult, error) {
	execOpts := &executeOptions{}
	for _, opt := range opts {
		opt(execOpts)
	}

	err := t.setPredecessors(ctx)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to set predecessors: %w", err)
	}

	op := t.proposal.Operations[index]

	// Get target contract
	execAddress := execOpts.callProxy
	if len(execAddress) == 0 {
		execAddress = t.proposal.TimelockAddresses[op.ChainSelector]
	}
	chainSpecificIndex := t.GetChainSpecificIndex(index)
	return t.executors[op.ChainSelector].Execute(
		ctx,
		op,
		execAddress,
		t.predecessors[op.ChainSelector][chainSpecificIndex],
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
