package canton

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	cselectors "github.com/smartcontractkit/chain-selectors"

	cantontypes "github.com/noders-team/go-daml/pkg/types"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client *client.DamlBindingClient
	userId string
	party  string
}

func NewConfigurer(client *client.DamlBindingClient, userId string, party string) (*Configurer, error) {
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

	// List known packages to find the package ID for mcms
	ListKnownPackagesResp, err := c.client.PackageMng.ListKnownPackages(ctx)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to list known packages: %w", err)
	}

	var mcmsPkgID string
	for _, p := range ListKnownPackagesResp {
		if strings.Contains(strings.ToLower(p.Name), "mcms") {
			mcmsPkgID = p.PackageID
			break
		}
	}
	if mcmsPkgID == "" {
		return types.TransactionResult{}, fmt.Errorf("failed to find mcms package")
	}

	commandID := uuid.Must(uuid.NewUUID()).String()
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "mcms-set-config",
			UserID:     c.userId,
			CommandID:  commandID,
			ActAs:      []string{c.party},
			Commands: []*model.Command{{
				Command: &model.ExerciseCommand{
					TemplateID: mcmsContract.GetTemplateID(),
					ContractID: exerciseCmd.ContractID,
					Choice:     exerciseCmd.Choice,
					Arguments:  exerciseCmd.Arguments,
				},
			}},
		},
	}

	submitResp, err := c.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to set config: %w", err)
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
