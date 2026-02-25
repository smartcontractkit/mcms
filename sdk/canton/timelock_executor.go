package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor executes scheduled Canton timelock operations (ExecuteScheduledBatch).
type TimelockExecutor struct {
	*TimelockInspector
	client apiv2.CommandServiceClient
	party  string
}

// NewTimelockExecutor creates a TimelockExecutor that submits ExecuteScheduledBatch via the given clients and party.
// timelockAddress (in Execute) is InstanceAddress hex; it is resolved to contract ID when submitting.
func NewTimelockExecutor(client apiv2.CommandServiceClient, stateClient apiv2.StateServiceClient, party string) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(client, stateClient, party),
		client:            client,
		party:             party,
	}
}

// Execute submits ExecuteScheduledBatch for the given batch operation (same opId hash as converter).
// timelockAddress is InstanceAddress hex; it is resolved to the current MCMS contract ID before submit.
func (t *TimelockExecutor) Execute(
	ctx context.Context,
	bop types.BatchOperation,
	timelockAddress string,
	predecessor common.Hash,
	salt common.Hash,
) (types.TransactionResult, error) {
	contractID, err := ResolveMCMSContractID(ctx, t.TimelockInspector.StateServiceClient(), t.party, timelockAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("resolve MCMS contract ID: %w", err)
	}

	if len(bop.Transactions) == 0 {
		return types.TransactionResult{}, fmt.Errorf("batch operation has no transactions")
	}

	calls := make([]mcms.TimelockCall, 0, len(bop.Transactions))
	callsForHash := make([]TimelockCallForHash, 0, len(bop.Transactions))
	var targetCids []string
	for _, tx := range bop.Transactions {
		var af AdditionalFields
		if err := json.Unmarshal(tx.AdditionalFields, &af); err != nil {
			return types.TransactionResult{}, fmt.Errorf("unmarshal transaction additional fields: %w", err)
		}
		calls = append(calls, mcms.TimelockCall{
			TargetInstanceId: cantontypes.TEXT(af.TargetInstanceId),
			FunctionName:     cantontypes.TEXT(af.FunctionName),
			OperationData:    cantontypes.TEXT(af.OperationData),
		})
		callsForHash = append(callsForHash, TimelockCallForHash{
			TargetInstanceId: af.TargetInstanceId,
			FunctionName:     af.FunctionName,
			OperationData:    af.OperationData,
		})
		targetCids = af.ContractIds
	}
	if len(targetCids) == 0 {
		targetCids = []string{contractID}
	}

	predecessorHex := hex.EncodeToString(predecessor[:])
	saltHex := hex.EncodeToString(salt[:])
	opIDStr := HashTimelockOpId(callsForHash, predecessorHex, saltHex)

	// Resolve InstanceAddress hex to current contract ID so Canton can parse them
	stateClient := t.TimelockInspector.StateServiceClient()
	targetCidSlice := make([]cantontypes.CONTRACT_ID, len(targetCids))
	for i, cid := range targetCids {
		resolved, err := ResolveContractIDIfInstanceAddress(ctx, stateClient, t.party, cid)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("resolve contract ID %q: %w", cid, err)
		}
		targetCidSlice[i] = cantontypes.CONTRACT_ID(resolved)
	}

	executeArgs := mcms.ExecuteScheduledBatch{
		Submitter:   cantontypes.PARTY(t.party),
		OpId:        cantontypes.TEXT(opIDStr),
		Calls:       calls,
		Predecessor: cantontypes.TEXT(predecessorHex),
		Salt:        cantontypes.TEXT(saltHex),
		TargetCids:  targetCidSlice,
	}

	mcmsContract := mcms.MCMS{}
	packageID, moduleName, entityName, err := parseTemplateIDFromString(mcmsContract.GetTemplateID())
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse template ID: %w", err)
	}

	commandID := uuid.Must(uuid.NewUUID()).String()
	req := &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			CommandId: commandID,
			ActAs:     []string{t.party},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Exercise{
					Exercise: &apiv2.ExerciseCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						ContractId:     contractID,
						Choice:         "ExecuteScheduledBatch",
						ChoiceArgument: ledger.MapToValue(executeArgs),
					},
				},
			}},
		},
	}

	resp, err := t.client.SubmitAndWaitForTransaction(ctx, req)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("submit ExecuteScheduledBatch: %w", err)
	}

	// Extract new MCMS contract ID from Created event (callers need it for subsequent resolution)
	newMCMSContractID := ""
	newMCMSTemplateID := ""
	transaction := resp.GetTransaction()
	for _, ev := range transaction.GetEvents() {
		if createdEv := ev.GetCreated(); createdEv != nil {
			templateID := formatTemplateID(createdEv.GetTemplateId())
			if NormalizeTemplateKey(templateID) == MCMSTemplateKey {
				newMCMSContractID = createdEv.GetContractId()
				newMCMSTemplateID = templateID
				break
			}
		}
	}
	if newMCMSContractID == "" {
		return types.TransactionResult{}, fmt.Errorf("ExecuteScheduledBatch tx had no Created MCMS event; refusing to continue with old CID=%s", contractID)
	}

	return types.TransactionResult{
		Hash:        commandID,
		ChainFamily: cselectors.FamilyCanton,
		RawData: map[string]any{
			"NewMCMSContractID": newMCMSContractID,
			"NewMCMSTemplateID": newMCMSTemplateID,
			"RawTx":             resp,
		},
	}, nil
}
