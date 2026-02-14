package canton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/google/uuid"

	"github.com/smartcontractkit/go-daml/pkg/service/ledger"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"

	"github.com/smartcontractkit/chainlink-canton/bindings"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
)

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector provides methods to query timelock state from Canton MCMS contracts.
type TimelockInspector struct {
	stateClient apiv2.StateServiceClient
	client      apiv2.CommandServiceClient
	userId      string
	party       string
}

// NewTimelockInspector creates a new TimelockInspector for Canton.
func NewTimelockInspector(stateClient apiv2.StateServiceClient, client apiv2.CommandServiceClient, userId, party string) *TimelockInspector {
	return &TimelockInspector{
		stateClient: stateClient,
		client:      client,
		userId:      userId,
		party:       party,
	}
}

// GetProposers returns the signer addresses for the Proposer role.
func (t *TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	mcmsContract, err := t.getMCMSContract(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCMS contract: %w", err)
	}
	return extractSignerAddresses(mcmsContract.Proposer.Config.Signers), nil
}

// GetExecutors is unsupported on Canton: there is no separate executor role.
func (t *TimelockInspector) GetExecutors(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("unsupported on Canton: no separate executor role")
}

// GetBypassers returns the signer addresses for the Bypasser role.
func (t *TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	mcmsContract, err := t.getMCMSContract(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCMS contract: %w", err)
	}
	return extractSignerAddresses(mcmsContract.Bypasser.Config.Signers), nil
}

// GetCancellers returns the signer addresses for the Canceller role.
func (t *TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	mcmsContract, err := t.getMCMSContract(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCMS contract: %w", err)
	}
	return extractSignerAddresses(mcmsContract.Canceller.Config.Signers), nil
}

// extractSignerAddresses extracts signer addresses from a slice of SignerInfo.
func extractSignerAddresses(signers []mcms.SignerInfo) []string {
	result := make([]string, len(signers))
	for i, s := range signers {
		result[i] = string(s.SignerAddress)
	}
	return result
}

// getMCMSContract queries the active MCMS contract by contract ID.
func (t *TimelockInspector) getMCMSContract(ctx context.Context, mcmsAddr string) (*mcms.MCMS, error) {
	// Get current ledger offset
	ledgerEndResp, err := t.stateClient.GetLedgerEnd(ctx, &apiv2.GetLedgerEndRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger end: %w", err)
	}

	// Query active contracts at current offset
	activeContractsResp, err := t.stateClient.GetActiveContracts(ctx, &apiv2.GetActiveContractsRequest{
		ActiveAtOffset: ledgerEndResp.GetOffset(),
		EventFormat: &apiv2.EventFormat{
			FiltersByParty: map[string]*apiv2.Filters{
				t.party: {
					Cumulative: []*apiv2.CumulativeFilter{
						{
							IdentifierFilter: &apiv2.CumulativeFilter_TemplateFilter{
								TemplateFilter: &apiv2.TemplateFilter{
									TemplateId: &apiv2.Identifier{
										PackageId:  "#mcms",
										ModuleName: "MCMS.Main",
										EntityName: "MCMS",
									},
									IncludeCreatedEventBlob: false,
								},
							},
						},
					},
				},
			},
			Verbose: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get active contracts: %w", err)
	}
	defer activeContractsResp.CloseSend()

	// Stream through active contracts to find the MCMS contract with matching ID
	for {
		resp, err := activeContractsResp.Recv()
		if errors.Is(err, io.EOF) {
			// Stream ended without finding the contract
			return nil, fmt.Errorf("MCMS contract with ID %s not found", mcmsAddr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to receive active contracts: %w", err)
		}

		activeContract, ok := resp.GetContractEntry().(*apiv2.GetActiveContractsResponse_ActiveContract)
		if !ok {
			continue
		}

		createdEvent := activeContract.ActiveContract.GetCreatedEvent()
		if createdEvent == nil {
			continue
		}

		// Check if contract ID matches
		if createdEvent.ContractId != mcmsAddr {
			continue
		}

		// Use bindings package to unmarshal the contract
		mcmsContract, err := bindings.UnmarshalActiveContract[mcms.MCMS](activeContract)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal MCMS contract: %w", err)
		}

		return mcmsContract, nil
	}
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

	// Parse template ID
	packageID, moduleName, entityName, err := parseTemplateIDFromString(mcmsContract.GetTemplateID())
	if err != nil {
		return 0, fmt.Errorf("failed to parse template ID: %w", err)
	}

	// Build choice argument using binding type
	input := mcms.GetMinDelay{
		Submitter: cantontypes.PARTY(t.party),
	}
	choiceArgument := ledger.MapToValue(input)

	// Submit the exercise command with LEDGER_EFFECTS shape to get exercise results
	resp, err := t.client.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "mcms-timelock-get-min-delay",
			CommandId:  fmt.Sprintf("GetMinDelay-%s", uuid.Must(uuid.NewUUID()).String()),
			ActAs:      []string{t.party},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Exercise{
					Exercise: &apiv2.ExerciseCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						ContractId:     address,
						Choice:         "GetMinDelay",
						ChoiceArgument: choiceArgument,
					},
				},
			}},
		},
		TransactionFormat: &apiv2.TransactionFormat{
			EventFormat: &apiv2.EventFormat{
				FiltersByParty: map[string]*apiv2.Filters{
					t.party: {},
				},
				Verbose: true,
			},
			TransactionShape: apiv2.TransactionShape_TRANSACTION_SHAPE_LEDGER_EFFECTS,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to exercise GetMinDelay: %w", err)
	}

	// Extract RelTime result (microseconds) from exercised event
	transaction := resp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if exercisedEv := event.GetExercised(); exercisedEv != nil && exercisedEv.GetChoice() == "GetMinDelay" {
			// RelTime in Daml is microseconds, convert to seconds
			result := exercisedEv.GetExerciseResult()
			if result != nil {
				// Try direct int64 value (covers both 0 and non-zero)
				if result.GetSum() != nil {
					if _, ok := result.GetSum().(*apiv2.Value_Int64); ok {
						return uint64(result.GetInt64() / 1_000_000), nil
					}
				}
				// Try record with microseconds field (Canton RELTIME format)
				if record := result.GetRecord(); record != nil {
					for _, field := range record.GetFields() {
						if field.GetLabel() == "microseconds" {
							return uint64(field.GetValue().GetInt64() / 1_000_000), nil
						}
					}
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

	packageID, moduleName, entityName, err := parseTemplateIDFromString(mcmsContract.GetTemplateID())
	if err != nil {
		return false, fmt.Errorf("failed to parse template ID: %w", err)
	}

	// Build choice argument using binding types based on choice name
	var choiceArgument *apiv2.Value
	switch choiceName {
	case "IsOperation":
		choiceArgument = ledger.MapToValue(mcms.IsOperation{
			Submitter: cantontypes.PARTY(t.party),
			OpId:      cantontypes.TEXT(opIdHex),
		})
	case "IsOperationPending":
		choiceArgument = ledger.MapToValue(mcms.IsOperationPending{
			Submitter: cantontypes.PARTY(t.party),
			OpId:      cantontypes.TEXT(opIdHex),
		})
	case "IsOperationReady":
		choiceArgument = ledger.MapToValue(mcms.IsOperationReady{
			Submitter: cantontypes.PARTY(t.party),
			OpId:      cantontypes.TEXT(opIdHex),
		})
	case "IsOperationDone":
		choiceArgument = ledger.MapToValue(mcms.IsOperationDone{
			Submitter: cantontypes.PARTY(t.party),
			OpId:      cantontypes.TEXT(opIdHex),
		})
	default:
		return false, fmt.Errorf("unsupported choice name: %s", choiceName)
	}

	// Submit the exercise command with LEDGER_EFFECTS shape to get exercise results for non-consuming choices
	resp, err := t.client.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: fmt.Sprintf("mcms-timelock-%s", choiceName),
			CommandId:  fmt.Sprintf("%s-%s-%d", choiceName, opIdHex[:16], time.Now().UnixNano()),
			ActAs:      []string{t.party},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Exercise{
					Exercise: &apiv2.ExerciseCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						ContractId:     address,
						Choice:         choiceName,
						ChoiceArgument: choiceArgument,
					},
				},
			}},
		},
		TransactionFormat: &apiv2.TransactionFormat{
			EventFormat: &apiv2.EventFormat{
				FiltersByParty: map[string]*apiv2.Filters{
					t.party: {},
				},
				Verbose: true,
			},
			TransactionShape: apiv2.TransactionShape_TRANSACTION_SHAPE_LEDGER_EFFECTS,
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to exercise %s: %w", choiceName, err)
	}

	// Extract boolean result from exercised event
	transaction := resp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if exercisedEv := event.GetExercised(); exercisedEv != nil && exercisedEv.GetChoice() == choiceName {
			result := exercisedEv.GetExerciseResult()
			if result != nil {
				return result.GetBool(), nil
			}
			return false, fmt.Errorf("exercised event found for %s but result is nil", choiceName)
		}
	}
	return false, fmt.Errorf("no exercised event found for %s (total events: %d)", choiceName, len(transaction.GetEvents()))
}
