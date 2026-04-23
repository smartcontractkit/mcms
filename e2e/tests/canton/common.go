//go:build e2e

package canton

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/mcms"
	"github.com/smartcontractkit/chainlink-canton/bindings/generated/mcms/mcmstest"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	"github.com/smartcontractkit/chainlink-canton/testhelpers"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"
	"github.com/smartcontractkit/go-daml/pkg/types"

	mcmscore "github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	e2e.TestSetup

	env testhelpers.TestEnvironment

	participant canton.Participant

	chainSelector       mcmstypes.ChainSelector
	chainId             int64
	packageIDs          []string
	mcmsInstanceAddress string // InstanceAddress hex (stable across SetRoot/ExecuteOp); use this everywhere instead of contract ID
	mcmsId              string // MCMS contract instanceId (base); DAML expects metadata.multisigId = makeMcmsId(instanceId, role) e.g. "mcms-test-001-proposer"
	proposerMcmsId      string // makeMcmsId(mcmsId, Proposer) for chain metadata and Op.multisigId
}

func (s *TestSuite) SetupSuite() {
	shared := GetSharedEnvironment(s.T())
	s.env = shared.Env
	s.participant = shared.Env.Chain.Participants[0]
	s.chainSelector = shared.ChainSelector
	s.packageIDs = shared.PackageIDs
}

const NumGroups = 32

func (s *TestSuite) DeployMCMSContract() {
	mcmsOwner := s.participant.PartyID
	chainId := int64(1)
	mcmsId := "mcms-" + uuid.New().String()[:8]

	mcmsInstanceAddr := s.createMCMS(s.T().Context(), s.participant, mcmsOwner, chainId, mcmsId, mcms.RoleProposer)
	s.mcmsInstanceAddress = mcmsInstanceAddr
	s.mcmsId = mcmsId
	// multisigId format: instanceId@partyId-role (see MCMS.Main.daml makeMcmsId)
	s.proposerMcmsId = fmt.Sprintf("%s@%s-proposer", mcmsId, mcmsOwner)
	s.chainId = chainId
}

func (s *TestSuite) DeployMCMSWithConfig(config *mcmstypes.Config) {
	s.DeployMCMSContract()

	// Set the config for all roles (proposer, canceller, bypasser) so tests can use any role
	roles := []cantonsdk.TimelockRole{
		cantonsdk.TimelockRoleProposer,
		cantonsdk.TimelockRoleCanceller,
		cantonsdk.TimelockRoleBypasser,
	}
	for _, role := range roles {
		configurer, err := cantonsdk.NewConfigurer(s.participant.LedgerServices.Command, s.participant.LedgerServices.State, s.participant.UserID, s.participant.PartyID, role)
		s.Require().NoError(err)

		_, err = configurer.SetConfig(s.T().Context(), s.mcmsInstanceAddress, config, true)
		s.Require().NoError(err)
	}
}

func (s *mcmsExecutorSetup) DeployCounterContract() {
	s.counterInstanceID = "counter-" + uuid.New().String()[:8]
	// DAML Counter.instanceId must NOT contain "@" - the full instanceAddress is calculated as instanceId@partyId

	// Create Counter contract
	counterContract := mcmstest.Counter{
		Owner:      types.PARTY(s.participant.PartyID),
		InstanceId: types.TEXT(s.counterInstanceID),
		Value:      types.INT64(0),
	}

	// Parse template ID
	exerciseCmd := counterContract.CreateCommand()
	packageID, moduleName, entityName, err := cantonsdk.ParseTemplateIDFromString(exerciseCmd.TemplateID)
	s.Require().NoError(err, "failed to parse template ID")

	// Convert create arguments to apiv2 format
	createArguments := ledger.ConvertToRecord(exerciseCmd.Arguments)

	commandID := uuid.Must(uuid.NewUUID()).String()
	submitResp, err := s.participant.LedgerServices.Command.SubmitAndWaitForTransaction(s.T().Context(), &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "counter-deploy",
			CommandId:  commandID,
			ActAs:      []string{s.participant.PartyID},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Create{
					Create: &apiv2.CreateCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						CreateArguments: createArguments,
					},
				},
			}},
		},
	})
	s.Require().NoError(err)

	// Extract contract ID
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			if templateID != "" {
				s.counterCID = createdEv.GetContractId()
				break
			}
		}
	}
	s.Require().NotEmpty(s.counterCID)
}

// createMCMS creates an MCMS contract and returns its InstanceAddress hex (stable reference for Canton).
func (s *TestSuite) createMCMS(ctx context.Context, participant canton.Participant, owner string, chainId int64, mcmsId string, role mcms.Role) string {
	// Create empty config
	emptyConfig := mcms.MultisigConfig{
		Signers:      []mcms.SignerInfo{},
		GroupQuorums: []types.INT64{types.INT64(1)},
		GroupParents: []types.INT64{types.INT64(1)},
	}

	// Create empty role state using zero values and nil for maps
	emptyRoleState := mcms.RoleState{
		Config:       emptyConfig,
		SeenHashes:   nil,
		ExpiringRoot: mcms.ExpiringRoot{},
		RootMetadata: mcms.RootMetadata{},
	}

	minDelayValue := &apiv2.Value{Sum: &apiv2.Value_Record{Record: &apiv2.Record{
		Fields: []*apiv2.RecordField{
			{Label: "microseconds", Value: &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: 0}}},
		},
	}}}

	// Create MCMS contract with new structure
	mcmsContract := mcms.MCMS{
		Owner:              types.PARTY(participant.PartyID),
		InstanceId:         types.TEXT(mcmsId),
		ChainId:            types.INT64(chainId),
		Proposer:           emptyRoleState,
		Canceller:          emptyRoleState,
		Bypasser:           emptyRoleState,
		BlockedFunctions:   nil,
		TimelockTimestamps: nil,
	}

	// Parse template ID
	exerciseCmd := mcmsContract.CreateCommand()
	packageID, moduleName, entityName, err := cantonsdk.ParseTemplateIDFromString(exerciseCmd.TemplateID)
	s.Require().NoError(err, "failed to parse template ID")

	// Convert create arguments to apiv2 format
	createArguments := ledger.ConvertToRecord(exerciseCmd.Arguments)

	// Remove minDelay from arguments
	filteredFields := make([]*apiv2.RecordField, 0, len(createArguments.Fields))
	for _, field := range createArguments.Fields {
		if field.Label != "minDelay" {
			filteredFields = append(filteredFields, field)
		}
	}
	createArguments.Fields = filteredFields

	// Submit via CommandService
	commandID := uuid.Must(uuid.NewUUID()).String()
	submitResp, err := participant.LedgerServices.Command.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "mcms-deploy",
			CommandId:  commandID,
			ActAs:      []string{participant.PartyID},
			Commands: []*apiv2.Command{{
				Command: &apiv2.Command_Create{
					Create: &apiv2.CreateCommand{
						TemplateId: &apiv2.Identifier{
							PackageId:  packageID,
							ModuleName: moduleName,
							EntityName: entityName,
						},
						CreateArguments: &apiv2.Record{Fields: append(
							createArguments.Fields,
							&apiv2.RecordField{Label: "minDelay", Value: minDelayValue},
						)},
					},
				},
			}},
		},
	})
	s.Require().NoError(err, "failed to submit MCMS deploy transaction")

	// Retrieve the contract ID and template ID from the create event
	mcmsContractID := ""
	mcmsTemplateID := ""
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalizedTemplateID := cantonsdk.NormalizeTemplateKey(templateID)
			if normalizedTemplateID == cantonsdk.MCMSTemplateKey {
				mcmsContractID = createdEv.GetContractId()
				mcmsTemplateID = templateID
				break
			}
		}
	}

	s.Require().NotEmpty(mcmsContractID, "failed to find MCMS contract in transaction events")
	s.Require().NotEmpty(mcmsTemplateID, "failed to find MCMS template ID in transaction events")

	// Return InstanceAddress hex so callers use a stable reference (contract ID changes after SetRoot/ExecuteOp)
	instanceAddress := contracts.InstanceID(mcmsId).RawInstanceAddress(types.PARTY(owner)).InstanceAddress()
	return instanceAddress.Hex()
}

func (s *TestSuite) SignProposal(proposal *mcmscore.Proposal, inspector sdk.Inspector, keys []*ecdsa.PrivateKey, quorum int) (*mcmscore.Signable, []mcmstypes.Signature, error) {
	inspectorsMap := map[mcmstypes.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}
	signable, err := mcmscore.NewSignable(proposal, inspectorsMap)
	if err != nil {
		return nil, nil, err
	}

	signatures := make([]mcmstypes.Signature, 0, quorum)
	for i := 0; i < len(keys) && i < quorum; i++ {
		sig, err := signable.SignAndAppend(mcmscore.NewPrivateKeySigner(keys[i]))
		if err != nil {
			return nil, nil, err
		}
		signatures = append(signatures, sig)
	}

	return signable, signatures, nil
}
