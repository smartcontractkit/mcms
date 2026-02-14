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
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor executes scheduled timelock operations on Canton MCMS contracts.
type TimelockExecutor struct {
	*TimelockInspector
	client apiv2.CommandServiceClient
	userId string
	party  string
}

// NewTimelockExecutor creates a new TimelockExecutor for Canton.
func NewTimelockExecutor(
	stateClient apiv2.StateServiceClient,
	client apiv2.CommandServiceClient,
	userId, party string,
) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(stateClient, client, userId, party),
		client:            client,
		userId:            userId,
		party:             party,
	}
}

// Execute executes a scheduled timelock batch operation.
// This exercises the ExecuteScheduledBatch choice on the MCMS contract.
func (t *TimelockExecutor) Execute(
	ctx context.Context,
	bop types.BatchOperation,
	timelockAddress string,
	predecessor common.Hash,
	salt common.Hash,
) (types.TransactionResult, error) {
	// Convert transactions to TimelockCall maps for the exercise command
	calls := make([]map[string]interface{}, len(bop.Transactions))
	targetCids := make([]interface{}, 0)

	timelockCalls := make([]TimelockCall, len(bop.Transactions))

	for i, tx := range bop.Transactions {
		var additionalFields AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			if err := json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
				return types.TransactionResult{}, fmt.Errorf("failed to unmarshal transaction additional fields: %w", err)
			}
		}

		// Use TargetInstanceId from AdditionalFields, or fall back to tx.To
		targetInstanceId := additionalFields.TargetInstanceId
		if targetInstanceId == "" {
			targetInstanceId = tx.To
		}

		// Use FunctionName from AdditionalFields
		functionName := additionalFields.FunctionName

		// Use OperationData from AdditionalFields, or hex-encode tx.Data
		operationData := additionalFields.OperationData
		if operationData == "" && len(tx.Data) > 0 {
			operationData = hex.EncodeToString(tx.Data)
		}

		calls[i] = map[string]interface{}{
			"targetInstanceId": targetInstanceId,
			"functionName":     functionName,
			"operationData":    operationData,
		}

		timelockCalls[i] = TimelockCall{
			TargetInstanceId: targetInstanceId,
			FunctionName:     functionName,
			OperationData:    operationData,
		}

		// Collect target CIDs for external calls
		if additionalFields.TargetCid != "" {
			targetCids = append(targetCids, additionalFields.TargetCid)
		}
	}

	// Convert predecessor and salt to hex strings
	predecessorHex := hex.EncodeToString(predecessor[:])
	saltHex := hex.EncodeToString(salt[:])

	// Compute opId
	opId := HashTimelockOpId(timelockCalls, predecessorHex, saltHex)
	opIdHex := hex.EncodeToString(opId[:])

	// Build exercise command manually since bindings don't have ExecuteScheduledBatch
	mcmsContract := mcms.MCMS{}

	// Parse template ID
	packageID, moduleName, entityName, err := parseTemplateIDFromString(mcmsContract.GetTemplateID())
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse template ID: %w", err)
	}

	// Convert calls to mcms.TimelockCall slice
	typedCalls := make([]mcms.TimelockCall, len(calls))
	for i, call := range calls {
		typedCalls[i] = mcms.TimelockCall{
			TargetInstanceId: cantontypes.TEXT(call["targetInstanceId"].(string)),
			FunctionName:     cantontypes.TEXT(call["functionName"].(string)),
			OperationData:    cantontypes.TEXT(call["operationData"].(string)),
		}
	}

	// Convert targetCids to typed slice
	typedTargetCids := make([]cantontypes.CONTRACT_ID, len(targetCids))
	for i, cid := range targetCids {
		typedTargetCids[i] = cantontypes.CONTRACT_ID(cid.(string))
	}

	// Build choice argument using binding type
	input := mcms.ExecuteScheduledBatch{
		Submitter:   cantontypes.PARTY(t.party),
		OpId:        cantontypes.TEXT(opIdHex),
		Calls:       typedCalls,
		Predecessor: cantontypes.TEXT(predecessorHex),
		Salt:        cantontypes.TEXT(saltHex),
		TargetCids:  typedTargetCids,
	}
	choiceArgument := ledger.MapToValue(input)

	// Generate command ID
	commandID := uuid.Must(uuid.NewUUID()).String()

	// Submit the exercise command
	submitResp, err := t.client.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "mcms-timelock-execute",
			CommandId:  commandID,
			ActAs:      []string{t.party},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Exercise{
					Exercise: &apiv2.ExerciseCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						ContractId:     timelockAddress,
						Choice:         "ExecuteScheduledBatch",
						ChoiceArgument: choiceArgument,
					},
				},
			}},
		},
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute scheduled batch: %w", err)
	}

	// Extract NEW MCMS CID from Created event
	newMCMSContractID := ""
	newMCMSTemplateID := ""
	transaction := submitResp.GetTransaction()
	for _, ev := range transaction.GetEvents() {
		if createdEv := ev.GetCreated(); createdEv != nil {
			templateID := formatTemplateID(createdEv.GetTemplateId())
			normalized := NormalizeTemplateKey(templateID)
			if normalized == MCMSTemplateKey {
				newMCMSContractID = createdEv.GetContractId()
				newMCMSTemplateID = templateID
				break
			}
		}
	}

	if newMCMSContractID == "" {
		return types.TransactionResult{}, fmt.Errorf("execute-scheduled-batch tx had no Created MCMS event; refusing to continue with old CID=%s", timelockAddress)
	}

	return types.TransactionResult{
		Hash:        commandID,
		ChainFamily: cselectors.FamilyCanton,
		RawData: map[string]any{
			"NewMCMSContractID": newMCMSContractID,
			"NewMCMSTemplateID": newMCMSTemplateID,
			"RawTx":             submitResp,
		},
	}, nil
}
