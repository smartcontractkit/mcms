//go:build e2e

package canton

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/noders-team/go-daml/pkg/client"
	"github.com/noders-team/go-daml/pkg/model"
	"github.com/noders-team/go-daml/pkg/types"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/chainlink-canton/contracts"
	"github.com/smartcontractkit/chainlink-canton/integration-tests/testhelpers"

	mcmscore "github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	e2e.TestSetup

	env testhelpers.TestEnvironment

	client      *client.DamlBindingClient
	participant testhelpers.Participant

	chainSelector  mcmstypes.ChainSelector
	chainId        int64
	packageIDs     []string
	mcmsContractID string
	mcmsId         string
}

func NewBindingClient(ctx context.Context, jwtToken, ledgerAPIURL, adminAPIURL string) (*client.DamlBindingClient, error) {
	bindingClient, err := client.NewDamlClient(jwtToken, ledgerAPIURL).
		WithAdminAddress(adminAPIURL).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create DAML binding client: %w", err)
	}

	return bindingClient, nil
}

func (s *TestSuite) SetupSuite() {
	s.T().Log("Spinning up Canton test environment...")
	s.env = testhelpers.NewTestEnvironment(s.T(), testhelpers.WithNumberOfParticipants(1))
	jwt, err := s.env.Participant(1).GetToken(s.T().Context())
	s.Require().NoError(err)
	participant := s.env.Participant(1)
	config := participant.GetConfig()
	s.client, err = NewBindingClient(s.T().Context(), jwt, config.GRPCLedgerAPIURL, config.AdminAPIURL)
	s.Require().NoError(err)
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
	baseMcmsId := "mcms-test-001"
	mcmsId := makeMcmsId(baseMcmsId, "proposer")

	mcmsContractId := s.createMCMS(s.T().Context(), s.participant, mcmsOwner, chainId, mcmsId, mcms.RoleProposer)
	s.mcmsContractID = mcmsContractId
	s.mcmsId = mcmsId
	s.chainId = chainId
}

func (s *TestSuite) createMCMS(ctx context.Context, participant testhelpers.Participant, owner string, chainId int64, mcmsId string, role mcms.Role) string {
	// Create empty expiring root
	emptyExpiringRoot := mcms.ExpiringRoot{
		Root:       types.TEXT(""),
		ValidUntil: types.TIMESTAMP(time.Unix(0, 0).UTC()),
		OpCount:    types.INT64(0),
	}

	// Create empty root metadata
	emptyRootMetadata := mcms.RootMetadata{
		ChainId:              types.INT64(0),
		MultisigId:           types.TEXT(""),
		PreOpCount:           types.INT64(0),
		PostOpCount:          types.INT64(0),
		OverridePreviousRoot: types.BOOL(false),
	}

	// Create MCMS contract
	mcmsContract := mcms.MCMS{
		Owner:      types.PARTY(participant.Party),
		InstanceId: types.TEXT(mcmsId),
		Role:       role,
		ChainId:    types.INT64(chainId),
		McmsId:     types.TEXT(mcmsId),
		Config: mcms.MultisigConfig{
			Signers:      []mcms.SignerInfo{},
			GroupQuorums: []types.INT64{types.INT64(1)},
			GroupParents: []types.INT64{types.INT64(1)},
		},
		SeenHashes:   types.GENMAP{}, // Empty map
		ExpiringRoot: emptyExpiringRoot,
		RootMetadata: emptyRootMetadata,
	}

	// Submit via binding client's CommandService
	commandID := uuid.Must(uuid.NewUUID()).String()
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "mcms-deploy",
			UserID:     participant.UserName,
			CommandID:  commandID,
			ActAs:      []string{participant.Party},
			Commands:   []*model.Command{{Command: mcmsContract.CreateCommand()}},
		},
	}

	submitResp, err := s.client.CommandService.SubmitAndWaitForTransaction(ctx, cmds)
	s.Require().NoError(err, "failed to submit MCMS deploy transaction")

	// Retrieve the contract ID and template ID from the create event
	mcmsContractID := ""
	mcmsTemplateID := ""
	for _, event := range submitResp.Transaction.Events {
		if event.Created == nil {
			continue
		}

		normalizedTemplateID := cantonsdk.NormalizeTemplateKey(event.Created.TemplateID)
		if normalizedTemplateID == cantonsdk.MCMSTemplateKey {
			mcmsContractID = event.Created.ContractID
			mcmsTemplateID = event.Created.TemplateID

			break
		}
	}

	s.Require().NotEmpty(mcmsContractID, "failed to find MCMS contract in transaction events")
	s.Require().NotEmpty(mcmsTemplateID, "failed to find MCMS template ID in transaction events")

	return mcmsContractID
}

// TODO: Use right role types
func makeMcmsId(baseId string, role string) string {
	return baseId + "-" + role
}

// TODO: Remove when validUntil calcualtion is considered in contracts
func (s *TestSuite) SignProposal(proposal *mcmscore.Proposal, signer mcmscore.PrivateKeySigner) mcmstypes.Signature {
	// Get the Merkle root of the proposal
	tree, err := proposal.MerkleTree()
	s.Require().NoError(err, "failed to calculate tree")

	root := tree.Root.Hex()
	// Get the valid until timestamp from the proposal metadata
	validUntil := proposal.ValidUntil

	// Compute the payload to sign (this should match what the MCMS contract expects)
	payload := ComputeSignedHash(root, validUntil)

	// Sign the payload
	sig, err := signer.Sign(payload)
	s.Require().NoError(err, "failed to sign proposal")

	signature, err := mcmstypes.NewSignatureFromBytes(sig)
	s.Require().NoError(err, "failed to create signature from bytes")

	// Append the signature to the proposal
	proposal.AppendSignature(signature)

	return signature
}

func ComputeSignedHash(root string, validUntil uint32) []byte {
	// Inner data: root || validUntil as hex
	validUntilHex := Uint32ToHex(validUntil)
	innerData, _ := hex.DecodeString(root + validUntilHex)
	innerHash := crypto.Keccak256(innerData)

	// EIP-191 prefix: "\x19Ethereum Signed Message:\n32"
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	prefixedData := append(prefix, innerHash...)

	return crypto.Keccak256Hash(prefixedData).Bytes()
}

// Uint32ToHex converts uint32 to 32-byte hex (matching Canton's placeholder)
// Canton currently uses padLeft32 "0" - we match that for compatibility
func Uint32ToHex(validUntil uint32) string {
	// For now, match Canton's placeholder implementation
	// TODO: Implement proper timestamp encoding when Canton updates
	return strings.Repeat("0", 64)
}
