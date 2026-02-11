//go:build e2e

package canton

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	"github.com/smartcontractkit/chainlink-canton/integration-tests/testhelpers"
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

	participant testhelpers.Participant

	chainSelector  mcmstypes.ChainSelector
	chainId        int64
	packageIDs     []string
	mcmsContractID string
	mcmsId         string
	proposerMcmsId string
}

func (s *TestSuite) SetupSuite() {
	s.T().Log("Spinning up Canton test environment...")
	s.env = testhelpers.NewTestEnvironment(s.T(), testhelpers.WithNumberOfParticipants(1))
	participant := s.env.Participant(1)
	s.participant = participant
	s.chainSelector = mcmstypes.ChainSelector(s.env.Chain.ChainSelector())
}

const NumGroups = 32

func (s *TestSuite) DeployMCMSContract() {
	s.T().Log("Uploading MCMS DAR...")

	mcmsDar, err := contracts.GetDar(contracts.MCMS, contracts.CurrentVersion)
	s.Require().NoError(err)

	packageIDs, err := testhelpers.UploadDARstoMultipleParticipants(s.T().Context(), [][]byte{mcmsDar}, s.participant)
	s.Require().NoError(err)
	s.packageIDs = packageIDs

	mcmsOwner := s.participant.Party
	chainId := int64(1)
	mcmsId := "mcms-test-001"

	mcmsContractId := s.createMCMS(s.T().Context(), s.participant, mcmsOwner, chainId, mcmsId, mcms.RoleProposer)
	s.mcmsContractID = mcmsContractId
	s.mcmsId = mcmsId
	s.proposerMcmsId = fmt.Sprintf("%s-%s", mcmsId, "proposer")
	s.chainId = chainId
}

func (s *TestSuite) DeployMCMSWithConfig(config *mcmstypes.Config) {
	s.DeployMCMSContract()

	// Set the config
	configurer, err := cantonsdk.NewConfigurer(s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)

	tx, err := configurer.SetConfig(s.T().Context(), s.mcmsContractID, config, true)
	s.Require().NoError(err)

	// Extract new contract ID
	rawData, ok := tx.RawData.(map[string]any)
	s.Require().True(ok)
	newContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newContractID
}

func (s *MCMSExecutorTestSuite) DeployCounterContract() {
	s.counterInstanceID = "counter-" + uuid.New().String()[:8]

	// Create Counter contract
	counterContract := mcms.Counter{
		Owner:      types.PARTY(s.participant.Party),
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
	submitResp, err := s.participant.CommandServiceClient.SubmitAndWaitForTransaction(s.T().Context(), &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "counter-deploy",
			CommandId:  commandID,
			ActAs:      []string{s.participant.Party},
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

func (s *TestSuite) createMCMS(ctx context.Context, participant testhelpers.Participant, owner string, chainId int64, mcmsId string, role mcms.Role) string {
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
		Owner:              types.PARTY(participant.Party),
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
	submitResp, err := participant.CommandServiceClient.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			WorkflowId: "mcms-deploy",
			CommandId:  commandID,
			ActAs:      []string{participant.Party},
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

	return mcmsContractID
}

// encodeSetConfigParams encodes SetConfig parameters for Canton MCMS
func EncodeSetConfigParams(s *TestSuite, signerAddresses []string, groupQuorums, groupParents []int64, clearRoot bool) string {
	var buf []byte

	// Encode signers list
	buf = append(buf, byte(len(signerAddresses))) // numSigners (1 byte)
	for i, signer := range signerAddresses {
		addrBytes, err := hex.DecodeString(signer)
		s.Require().NoError(err, "failed to decode signer address hex")
		buf = append(buf, byte(len(addrBytes))) // addressLen (1 byte)
		buf = append(buf, addrBytes...)         // address bytes

		// SignerIndex (4 bytes, big-endian)
		indexBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(indexBytes, uint32(i)) //nolint:gosec
		buf = append(buf, indexBytes...)

		// SignerGroup (4 bytes, big-endian)
		groupBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(groupBytes, uint32(0)) //nolint:gosec
		buf = append(buf, groupBytes...)
	}

	// Encode group quorums
	buf = append(buf, byte(len(groupQuorums))) // numQuorums (1 byte)
	for _, quorum := range groupQuorums {
		quorumBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(quorumBytes, uint32(quorum)) //nolint:gosec
		buf = append(buf, quorumBytes...)
	}

	// Encode group parents
	buf = append(buf, byte(len(groupParents))) // numParents (1 byte)
	for _, parent := range groupParents {
		parentBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(parentBytes, uint32(parent)) //nolint:gosec
		buf = append(buf, parentBytes...)
	}

	// Encode clearRoot (1 byte)
	if clearRoot {
		buf = append(buf, 0x01)
	} else {
		buf = append(buf, 0x00)
	}

	return hex.EncodeToString(buf)
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
