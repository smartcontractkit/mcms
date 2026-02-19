package canton

import (
	"context"
	"errors"

	"github.com/smartcontractkit/mcms/sdk"
)

const errMsgUnsupported = "unsupported on Canton"

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector is a stub implementation for Canton timelock inspection.
// All methods return "unsupported on Canton" until Phase C implementation.
type TimelockInspector struct{}

// NewTimelockInspector creates a new TimelockInspector (stub).
func NewTimelockInspector() *TimelockInspector {
	return &TimelockInspector{}
}

func (t *TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return false, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return false, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return false, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return false, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetMinDelay(ctx context.Context, address string) (uint64, error) {
	return 0, errors.New(errMsgUnsupported)
}
