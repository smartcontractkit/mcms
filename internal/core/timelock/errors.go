package timelock

import (
	"fmt"
)

type InvalidDelayError struct {
	Delay string
}

func (e *InvalidDelayError) Error() string {
	return fmt.Sprintf("invalid min delay: %s", e.Delay)
}

type InvalidTimelockOperationError struct {
	ReceivedTimelockOperation string
}

func (e *InvalidTimelockOperationError) Error() string {
	return fmt.Sprintf("invalid timelock operation: %s", e.ReceivedTimelockOperation)
}

type NoTransactionsError struct{}

func (e *NoTransactionsError) Error() string {
	return "no transactions"
}

type InvalidOperationError struct {
	Operation string
}

func (e *InvalidOperationError) Error() string {
	return fmt.Sprintf("invalid timelock operation: %s", e.Operation)
}
