package canton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor executes scheduled timelock operations on Canton MCMS contracts.
type TimelockExecutor struct {
	*TimelockInspector
	client *client.DamlBindingClient
	userId string
	party  string
}

// NewTimelockExecutor creates a new TimelockExecutor for Canton.
func NewTimelockExecutor(
	stateClient apiv2.StateServiceClient,
	client *client.DamlBindingClient,
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
	exerciseCmd := &model.ExerciseCommand{
		TemplateID: mcmsContract.GetTemplateID(),
		ContractID: timelockAddress,
		Choice:     "ExecuteScheduledBatch",
		Arguments: map[string]interface{}{
			"submitter":   t.party,
			"opId":        opIdHex,
			"calls":       calls,
			"predecessor": predecessorHex,
			"salt":        saltHex,
			"targetCids":  targetCids,
		},
	}

	// Generate command ID
	commandID := uuid.Must(uuid.NewUUID()).String()

	// Submit the exercise command
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "mcms-timelock-execute",
			UserID:     t.userId,
			CommandID:  commandID,
			ActAs:      []string{t.party},
			Commands: []*model.Command{{
				Command: exerciseCmd,
			}},
		},
	}

	submitResp, err := t.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to execute scheduled batch: %w", err)
	}

	// Extract NEW MCMS CID from Created event
	newMCMSContractID := ""
	newMCMSTemplateID := ""
	for _, ev := range submitResp.Transaction.Events {
		if ev.Created == nil {
			continue
		}
		normalized := NormalizeTemplateKey(ev.Created.TemplateID)
		if normalized == MCMSTemplateKey {
			newMCMSContractID = ev.Created.ContractID
			newMCMSTemplateID = ev.Created.TemplateID
			break
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
