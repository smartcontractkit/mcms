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
	client        apiv2.CommandServiceClient
	party         string
	mcmsPackageID string // empty => use mcms.PackageName
}

// NewTimelockExecutor creates a TimelockExecutor that submits ExecuteScheduledBatch via the given client and party.
// mcmsPackageID is optional; if empty, the bindings' default package name is used for the template ID.
func NewTimelockExecutor(client apiv2.CommandServiceClient, party string, mcmsPackageID string) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(client, party, mcmsPackageID),
		client:            client,
		party:             party,
		mcmsPackageID:     mcmsPackageID,
	}
}

// Execute submits ExecuteScheduledBatch for the given batch operation (same opId hash as converter).
func (t *TimelockExecutor) Execute(
	ctx context.Context,
	bop types.BatchOperation,
	timelockAddress string,
	predecessor common.Hash,
	salt common.Hash,
) (types.TransactionResult, error) {
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
		targetCids = []string{timelockAddress}
	}

	predecessorHex := hex.EncodeToString(predecessor[:])
	saltHex := hex.EncodeToString(salt[:])
	opIDStr := HashTimelockOpId(callsForHash, predecessorHex, saltHex)

	targetCidSlice := make([]cantontypes.CONTRACT_ID, len(targetCids))
	for i, cid := range targetCids {
		targetCidSlice[i] = cantontypes.CONTRACT_ID(cid)
	}

	executeArgs := mcms.ExecuteScheduledBatch{
		Submitter:   cantontypes.PARTY(t.party),
		OpId:        cantontypes.TEXT(opIDStr),
		Calls:       calls,
		Predecessor: cantontypes.TEXT(predecessorHex),
		Salt:        cantontypes.TEXT(saltHex),
		TargetCids:  targetCidSlice,
	}

	pkgID := t.mcmsPackageID
	if pkgID == "" {
		pkgID = mcms.PackageName
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
							PackageId:  pkgID,
							ModuleName: "MCMS.Main",
							EntityName: "MCMS",
						},
						ContractId:     timelockAddress,
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

	return types.TransactionResult{
		Hash:        commandID,
		ChainFamily: cselectors.FamilyCanton,
		RawData:     map[string]any{"RawTx": resp},
	}, nil
}
