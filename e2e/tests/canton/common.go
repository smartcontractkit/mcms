//go:build e2e

package canton

import (
	"context"

	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-canton-internal/contracts"
	"github.com/smartcontractkit/chainlink-canton-internal/integration-tests/testhelpers"
	apiv2 "github.com/smartcontractkit/chainlink-canton-internal/pb/gen/com/daml/ledger/api/v2"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	e2e.TestSetup

	env testhelpers.TestEnvironment
}

func (s *TestSuite) SetupSuite() {
	s.T().Log("Spinning up Canton test environment...")
	s.env = testhelpers.NewTestEnvironment(s.T(), testhelpers.WithNumberOfParticipants(1))
}

const NumGroups = 32

func (s *TestSuite) DeployMCMSContract() {

	participant := s.env.Participant(1)

	// ========================
	// |   Setup: Upload DAR  |
	// ========================

	s.T().Log("Uploading MCMS DAR...")

	mcmsDar, err := contracts.GetDar(contracts.MCMS, contracts.CurrentVersion)
	s.Require().NoError(err)

	packageIDs, err := testhelpers.UploadDARstoMultipleParticipants(s.T().Context(), [][]byte{mcmsDar}, participant)
	s.Require().NoError(err)
	s.T().Logf("Uploaded MCMS DAR, package IDs: %v", packageIDs)

	mcmsOwner := participant.Party

	chainId := 1
	baseMcmsId := "mcms-test-001"
	mcmsId := makeMcmsId(baseMcmsId, "proposer")

	mcmsContractId, err := s.createMCMS(s.T().Context(), participant, mcmsOwner, chainId, mcmsId)
	s.Require().NoError(err)
	s.T().Logf("Created MCMS contract with ID: %s", mcmsContractId)
}

func (s *TestSuite) createMCMS(ctx context.Context, participant testhelpers.Participant, owner string, chainId int, mcmsId string) (string, error) {
	emptyMap := &apiv2.Value{Sum: &apiv2.Value_GenMap{GenMap: &apiv2.GenMap{Entries: []*apiv2.GenMap_Entry{}}}}
	epochTime := &apiv2.Value{Sum: &apiv2.Value_Timestamp{Timestamp: 0}}
	emptyExpiringRoot := &apiv2.Value{Sum: &apiv2.Value_Record{Record: &apiv2.Record{
		Fields: []*apiv2.RecordField{
			{Label: "root", Value: &apiv2.Value{Sum: &apiv2.Value_Text{Text: ""}}},
			{Label: "validUntil", Value: epochTime},
			{Label: "opCount", Value: &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: 0}}},
		},
	}}}

	createRes, err := participant.CommandServiceClient.SubmitAndWaitForTransaction(ctx, &apiv2.SubmitAndWaitForTransactionRequest{
		Commands: &apiv2.Commands{
			CommandId: uuid.New().String(),
			Commands: []*apiv2.Command{
				{
					Command: &apiv2.Command_Create{
						Create: &apiv2.CreateCommand{
							TemplateId: &apiv2.Identifier{
								PackageId:  "#mcms",
								ModuleName: "MCMS.Main",
								EntityName: "MCMS",
							},
							CreateArguments: &apiv2.Record{Fields: []*apiv2.RecordField{
								{Label: "owner", Value: &apiv2.Value{Sum: &apiv2.Value_Party{Party: owner}}},
								{Label: "role", Value: &apiv2.Value{Sum: &apiv2.Value_Enum{Enum: &apiv2.Enum{Constructor: "Proposer"}}}},
								{Label: "chainId", Value: &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: int64(chainId)}}},
								{Label: "mcmsId", Value: &apiv2.Value{Sum: &apiv2.Value_Text{Text: mcmsId}}},
								{Label: "config", Value: &apiv2.Value{Sum: &apiv2.Value_Record{Record: &apiv2.Record{
									Fields: []*apiv2.RecordField{
										{Label: "signers", Value: &apiv2.Value{Sum: &apiv2.Value_List{List: &apiv2.List{Elements: []*apiv2.Value{}}}}},
										{Label: "groupQuorums", Value: &apiv2.Value{Sum: &apiv2.Value_List{List: &apiv2.List{Elements: makeInt64List(NumGroups, 0)}}}},
										{Label: "groupParents", Value: &apiv2.Value{Sum: &apiv2.Value_List{List: &apiv2.List{Elements: makeInt64List(NumGroups, 0)}}}},
									},
								}}}},
								{Label: "seenHashes", Value: emptyMap},
								{Label: "expiringRoot", Value: emptyExpiringRoot},
								{Label: "rootMetadata", Value: &apiv2.Value{Sum: &apiv2.Value_Record{Record: &apiv2.Record{Fields: []*apiv2.RecordField{
									{Label: "chainId", Value: &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: 0}}},
									{Label: "multisigId", Value: &apiv2.Value{Sum: &apiv2.Value_Text{Text: ""}}},
									{Label: "preOpCount", Value: &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: 0}}},
									{Label: "postOpCount", Value: &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: 0}}},
									{Label: "overridePreviousRoot", Value: &apiv2.Value{Sum: &apiv2.Value_Bool{Bool: false}}},
								}}}}},
							}},
						},
					},
				},
			},
			ActAs: []string{owner},
		},
	})
	s.Require().NoError(err)

	return createRes.GetTransaction().GetEvents()[0].GetCreated().GetContractId(), nil
}

func makeInt64List(count int, value int64) []*apiv2.Value {
	result := make([]*apiv2.Value, count)
	for i := 0; i < count; i++ {
		result[i] = &apiv2.Value{Sum: &apiv2.Value_Int64{Int64: value}}
	}
	return result
}

// TODO: Use right role types
func makeMcmsId(baseId string, role string) string {
	return baseId + "-" + role
}
