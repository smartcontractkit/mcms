package canton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/google/uuid"
	"github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector provides methods to query timelock state from Canton MCMS contracts.
// Canton uses party-based access control instead of role-based, so GetProposers, GetExecutors,
// GetBypassers, and GetCancellers return "unsupported" errors.
type TimelockInspector struct {
	stateClient apiv2.StateServiceClient
	client      *client.DamlBindingClient
	userId      string
	party       string
}

// NewTimelockInspector creates a new TimelockInspector for Canton.
func NewTimelockInspector(stateClient apiv2.StateServiceClient, client *client.DamlBindingClient, userId, party string) *TimelockInspector {
	return &TimelockInspector{
		stateClient: stateClient,
		client:      client,
		userId:      userId,
		party:       party,
	}
}

// TODO: Regenerate MCMS bindings to get latest MCMS state
func (t *TimelockInspector) GetProposers(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("TODO: Regenerate MCMS bindings to get latest MCMS state")
}

// TODO: Regenerate MCMS bindings to get latest MCMS state
func (t *TimelockInspector) GetExecutors(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("TODO: Regenerate MCMS bindings to get latest MCMS state")
}

// TODO: Regenerate MCMS bindings to get latest MCMS state
func (t *TimelockInspector) GetBypassers(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("TODO: Regenerate MCMS bindings to get latest MCMS state")
}

// TODO: Regenerate MCMS bindings to get latest MCMS state
func (t *TimelockInspector) GetCancellers(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("TODO: Regenerate MCMS bindings to get latest MCMS state")
}

// IsOperation checks if an operation exists in the timelock.
func (t *TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseTimelockViewChoice(ctx, address, "IsOperation", opID)
}

// IsOperationPending checks if an operation is pending (scheduled but not done).
func (t *TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseTimelockViewChoice(ctx, address, "IsOperationPending", opID)
}

// IsOperationReady checks if an operation is ready (delay passed, not done).
func (t *TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseTimelockViewChoice(ctx, address, "IsOperationReady", opID)
}

// IsOperationDone checks if an operation has been executed.
func (t *TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseTimelockViewChoice(ctx, address, "IsOperationDone", opID)
}

// GetMinDelay returns the minimum delay for scheduled operations in seconds.
func (t *TimelockInspector) GetMinDelay(ctx context.Context, address string) (uint64, error) {
	// Build exercise command for GetMinDelay view choice
	mcmsContract := mcms.MCMS{}
	exerciseCmd := &model.ExerciseCommand{
		TemplateID: mcmsContract.GetTemplateID(),
		ContractID: address,
		Choice:     "GetMinDelay",
		Arguments: map[string]interface{}{
			"submitter": t.party,
		},
	}

	// Submit the exercise command
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "mcms-timelock-get-min-delay",
			UserID:     t.userId,
			CommandID:  fmt.Sprintf("GetMinDelay-%s", uuid.Must(uuid.NewUUID()).String()),
			ActAs:      []string{t.party},
			Commands: []*model.Command{{
				Command: exerciseCmd,
			}},
		},
	}

	// Use SubmitAndWaitForTransaction and read the result from events
	resp, err := t.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	if err != nil {
		return 0, fmt.Errorf("failed to exercise GetMinDelay: %w", err)
	}

	// Extract RelTime result (microseconds) from exercised event
	for _, event := range resp.Transaction.Events {
		if event.Exercised != nil && event.Exercised.Choice == "GetMinDelay" {
			// RelTime in Daml is microseconds, convert to seconds
			switch v := event.Exercised.ExerciseResult.(type) {
			case float64:
				return uint64(v / 1_000_000), nil
			case int64:
				return uint64(v / 1_000_000), nil
			case map[string]interface{}:
				if micros, ok := v["microseconds"].(float64); ok {
					return uint64(micros / 1_000_000), nil
				}
			}
		}
	}
	return 0, fmt.Errorf("no exercise result found for GetMinDelay")
}

// exerciseTimelockViewChoice exercises a timelock view choice and returns the boolean result.
func (t *TimelockInspector) exerciseTimelockViewChoice(ctx context.Context, address, choiceName string, opID [32]byte) (bool, error) {
	// Convert opID to hex string for Canton TEXT type
	opIdHex := hex.EncodeToString(opID[:])

	// Build exercise command
	mcmsContract := mcms.MCMS{}
	exerciseCmd := &model.ExerciseCommand{
		TemplateID: mcmsContract.GetTemplateID(),
		ContractID: address,
		Choice:     choiceName,
		Arguments: map[string]interface{}{
			"submitter": t.party,
			"opId":      opIdHex,
		},
	}

	// Submit the exercise command
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: fmt.Sprintf("mcms-timelock-%s", choiceName),
			UserID:     t.userId,
			CommandID:  fmt.Sprintf("%s-%s-%d", choiceName, opIdHex[:16], time.Now().UnixNano()),
			ActAs:      []string{t.party},
			Commands: []*model.Command{{
				Command: exerciseCmd,
			}},
		},
	}

	resp, err := t.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	if err != nil {
		return false, fmt.Errorf("failed to exercise %s: %w", choiceName, err)
	}

	// Extract boolean result from exercised event
	for _, event := range resp.Transaction.Events {
		if event.Exercised != nil && event.Exercised.Choice == choiceName {
			if result, ok := event.Exercised.ExerciseResult.(bool); ok {
				return result, nil
			}
		}
	}
	return false, fmt.Errorf("no exercise result found for %s", choiceName)
}
