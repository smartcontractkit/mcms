package canton

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/google/uuid"
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
)

const errMsgUnsupported = "unsupported on Canton"

var _ sdk.TimelockInspector = (*TimelockInspector)(nil)

// TimelockInspector inspects Canton timelock state via MCMS read-only choices
// (IsOperation, IsOperationPending, IsOperationReady, IsOperationDone, GetMinDelay).
// Role lists (GetProposers, etc.) return "unsupported on Canton" like Aptos.
// address parameters are InstanceAddress hex (Canton); they are resolved to contract ID when exercising.
type TimelockInspector struct {
	client        apiv2.CommandServiceClient
	stateClient   apiv2.StateServiceClient
	party         string
	mcmsPackageID string
}

// NewTimelockInspector creates a TimelockInspector that queries the ledger via the given clients.
// mcmsPackageID may be empty to use the bindings' default package name.
func NewTimelockInspector(client apiv2.CommandServiceClient, stateClient apiv2.StateServiceClient, party string, mcmsPackageID string) *TimelockInspector {
	return &TimelockInspector{
		client:        client,
		stateClient:   stateClient,
		party:         party,
		mcmsPackageID: mcmsPackageID,
	}
}

// StateServiceClient returns the state service client for resolution (InstanceAddress to contract ID).
func (t *TimelockInspector) StateServiceClient() apiv2.StateServiceClient {
	return t.stateClient
}

func (t *TimelockInspector) GetProposers(ctx context.Context, address string) ([]string, error) {
	// TODO: implement this method
	_ = address
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetExecutors(ctx context.Context, address string) ([]string, error) {
	// TODO: implement this method
	_ = address
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetBypassers(ctx context.Context, address string) ([]string, error) {
	// TODO: implement this method
	_ = address
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) GetCancellers(ctx context.Context, address string) ([]string, error) {
	// TODO: implement this method
	_ = address
	return nil, errors.New(errMsgUnsupported)
}

func (t *TimelockInspector) IsOperation(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseBoolChoice(ctx, address, "IsOperation", opID)
}

func (t *TimelockInspector) IsOperationPending(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseBoolChoice(ctx, address, "IsOperationPending", opID)
}

func (t *TimelockInspector) IsOperationReady(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseBoolChoice(ctx, address, "IsOperationReady", opID)
}

func (t *TimelockInspector) IsOperationDone(ctx context.Context, address string, opID [32]byte) (bool, error) {
	return t.exerciseBoolChoice(ctx, address, "IsOperationDone", opID)
}

func (t *TimelockInspector) GetMinDelay(ctx context.Context, address string) (uint64, error) {
	contractID, err := ResolveMCMSContractID(ctx, t.stateClient, t.party, address)
	if err != nil {
		return 0, fmt.Errorf("resolve MCMS contract ID: %w", err)
	}
	pkgID := t.mcmsPackageID
	if pkgID == "" {
		pkgID = mcms.PackageName
	}
	args := mcms.GetMinDelay{Submitter: cantontypes.PARTY(t.party)}
	req := t.exerciseRequest(pkgID, contractID, "GetMinDelay", ledger.MapToValue(args))
	resp, err := t.client.SubmitAndWaitForTransaction(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("GetMinDelay: %w", err)
	}
	events := resp.GetTransaction().GetEvents()
	if len(events) == 0 {
		return 0, fmt.Errorf("GetMinDelay: no events in transaction")
	}
	ex := events[0].GetExercised()
	if ex == nil {
		return 0, fmt.Errorf("GetMinDelay: first event is not exercise")
	}
	// GetMinDelay returns RelTime = record with "microseconds" field
	rec := ex.GetExerciseResult().GetRecord()
	if rec == nil || len(rec.GetFields()) == 0 {
		return 0, fmt.Errorf("GetMinDelay: result is not a record with fields")
	}
	// first field is "microseconds" (Int64)
	val := rec.GetFields()[0].GetValue()
	if val == nil {
		return 0, fmt.Errorf("GetMinDelay: missing microseconds value")
	}
	us := val.GetInt64()
	if us < 0 {
		return 0, fmt.Errorf("GetMinDelay: invalid microseconds %d", us)
	}
	return uint64(us / 1_000_000), nil
}

func (t *TimelockInspector) exerciseBoolChoice(ctx context.Context, address string, choice string, opID [32]byte) (bool, error) {
	contractID, err := ResolveMCMSContractID(ctx, t.stateClient, t.party, address)
	if err != nil {
		return false, fmt.Errorf("resolve MCMS contract ID: %w", err)
	}
	pkgID := t.mcmsPackageID
	if pkgID == "" {
		pkgID = mcms.PackageName
	}
	opIDStr := hex.EncodeToString(opID[:])
	party := cantontypes.PARTY(t.party)
	var choiceArg *apiv2.Value
	switch choice {
	case "IsOperation":
		choiceArg = ledger.MapToValue(mcms.IsOperation{Submitter: party, OpId: cantontypes.TEXT(opIDStr)})
	case "IsOperationPending":
		choiceArg = ledger.MapToValue(mcms.IsOperationPending{Submitter: party, OpId: cantontypes.TEXT(opIDStr)})
	case "IsOperationReady":
		choiceArg = ledger.MapToValue(mcms.IsOperationReady{Submitter: party, OpId: cantontypes.TEXT(opIDStr)})
	case "IsOperationDone":
		choiceArg = ledger.MapToValue(mcms.IsOperationDone{Submitter: party, OpId: cantontypes.TEXT(opIDStr)})
	default:
		return false, fmt.Errorf("unknown choice %s", choice)
	}
	req := t.exerciseRequest(pkgID, contractID, choice, choiceArg)
	resp, err := t.client.SubmitAndWaitForTransaction(ctx, req)
	if err != nil {
		return false, fmt.Errorf("%s: %w", choice, err)
	}
	events := resp.GetTransaction().GetEvents()
	if len(events) == 0 {
		return false, fmt.Errorf("%s: no events", choice)
	}
	ex := events[0].GetExercised()
	if ex == nil {
		return false, fmt.Errorf("%s: first event is not exercise", choice)
	}
	return valueToBool(ex.GetExerciseResult())
}

func (t *TimelockInspector) exerciseRequest(pkgID, contractID, choice string, choiceArg *apiv2.Value) *apiv2.SubmitAndWaitForTransactionRequest {
	return &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			CommandId: uuid.Must(uuid.NewUUID()).String(),
			ActAs:     []string{t.party},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Exercise{
					Exercise: &apiv2.ExerciseCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  pkgID,
							ModuleName: "MCMS.Main",
							EntityName: "MCMS",
						},
						ContractId:     contractID,
						Choice:         choice,
						ChoiceArgument: choiceArg,
					},
				},
			}},
		},
		TransactionFormat: &apiv2.TransactionFormat{
			TransactionShape: apiv2.TransactionShape_TRANSACTION_SHAPE_LEDGER_EFFECTS,
		},
	}
}

func valueToBool(v *apiv2.Value) (bool, error) {
	if v == nil {
		return false, errors.New("nil value")
	}
	switch s := v.Sum.(type) {
	case *apiv2.Value_Bool:
		return s.Bool, nil
	case *apiv2.Value_Variant:
		// Daml Bool is sometimes encoded as variant True | False
		if s.Variant != nil {
			c := s.Variant.Constructor
			if c == "True" {
				return true, nil
			}
			if c == "False" {
				return false, nil
			}
		}
	}
	return false, fmt.Errorf("value is not Bool or Bool variant: %T", v.Sum)
}
