package timelock

import "github.com/smartcontractkit/mcms/types"

// TimelockEncoder is an interface that all chain timelock proposals must implement
type TimelockEncoder interface {
	// Converts the proposal into the chain specific transaction data format the signer needs
	Encode() ([]types.Operation, error)
}

type TimelockOperation string

const (
	Schedule TimelockOperation = "schedule"
	Cancel   TimelockOperation = "cancel"
	Bypass   TimelockOperation = "bypass"
)

// Chain agnostic configuration for a timelock proposal. These proposals target a timelock contract per chain and has no context about the signer (EOA, MCMS, etc)
type TimelockProposal struct {
	// What the Timelock will perform on the transactions specified
	Operation TimelockOperation `json:"operation"` // Always 'schedule', 'cancel', or 'bypass'

	// List of batches to be operated from the Timelock contract
	Batches []types.BatchChainOperation

	// MinDelay is the time duration for the timelock to wait before executing the transaction (only useful when scheduling)
	// Q: Format ? (1d, 1w, 1m, 1y, null)
	// Q: Why minDelay and not delay? MinDelay could be confused with the Timelock configured minimum delay
	MinDelay string `json:"minDelay"`
}

func NewTimelockProposal(operation TimelockOperation, batches []types.BatchChainOperation, delay string) (*TimelockProposal, error) {
	t := TimelockProposal{
		Operation: operation,
		Batches:   batches,
		MinDelay:  delay,
	}

	if err := t.Validate(); err != nil {
		return nil, err
	}

	return &t, nil
}

func (t TimelockProposal) Validate() error {
	if t.Operation == "" {
		return &InvalidOperationError{}
	}

	if t.MinDelay == "" {
		return &InvalidDelayError{}
	}

	if len(t.Batches) == 0 {
		return &NoTransactionsError{}
	}

	return nil
}
