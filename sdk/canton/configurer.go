package canton

import (
	"context"
	"fmt"
	"strings"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/google/uuid"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	cantontypes "github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client apiv2.CommandServiceClient
	userId string
	party  string
}

func NewConfigurer(client apiv2.CommandServiceClient, userId string, party string) (*Configurer, error) {
	return &Configurer{
		client: client,
		userId: userId,
		party:  party,
	}, nil
}

func (c Configurer) SetConfig(ctx context.Context, mcmsAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	groupQuorum, groupParents, signerAddresses, signerGroups, err := sdk.ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	signers := make([]mcms.SignerInfo, len(signerAddresses))
	for i, addr := range signerAddresses {
		addrStr := strings.ToLower(addr.String())
		addrStr = strings.TrimPrefix(addrStr, "0x")
		signers[i] = mcms.SignerInfo{
			SignerAddress: cantontypes.TEXT(addrStr),
			SignerGroup:   cantontypes.INT64(signerGroups[i]),
			SignerIndex:   cantontypes.INT64(i),
		}
	}

	groupQuorumsTyped := make([]cantontypes.INT64, len(groupQuorum))
	for i, q := range groupQuorum {
		groupQuorumsTyped[i] = cantontypes.INT64(q)
	}

	groupParentsTyped := make([]cantontypes.INT64, len(groupParents))
	for i, p := range groupParents {
		groupParentsTyped[i] = cantontypes.INT64(p)
	}

	input := mcms.SetConfig{
		NewSigners:      signers,
		NewGroupQuorums: groupQuorumsTyped,
		NewGroupParents: groupParentsTyped,
		ClearRoot:       cantontypes.BOOL(clearRoot),
	}
	// Build exercise command using generated bindings
	mcmsContract := mcms.MCMS{}
	exerciseCmd := mcmsContract.SetConfig(mcmsAddr, input)

	// Parse template ID
	packageID, moduleName, entityName, err := parseTemplateIDFromString(mcmsContract.GetTemplateID())
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse template ID: %w", err)
	}

	// Convert input to choice argument
	choiceArgument := ledger.MapToValue(input)

	commandID := uuid.Must(uuid.NewUUID()).String()
	submitResp, err := c.client.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "mcms-set-config",
			CommandId:  commandID,
			ActAs:      []string{c.party},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Exercise{
					Exercise: &apiv2.ExerciseCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						ContractId:     exerciseCmd.ContractID,
						Choice:         exerciseCmd.Choice,
						ChoiceArgument: choiceArgument,
					},
				},
			}},
		},
	})
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
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
		return types.TransactionResult{}, fmt.Errorf("set-config tx had no Created MCMS event; refusing to continue with old CID=%s", mcmsAddr)
	}

	return types.TransactionResult{
		Hash:        "tx.Digest",
		ChainFamily: cselectors.FamilyCanton,
		RawData: map[string]any{
			"NewMCMSContractID": newMCMSContractID,
			"NewMCMSTemplateID": newMCMSTemplateID,
			"RawTx":             submitResp,
		},
	}, nil

}
