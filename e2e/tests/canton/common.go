//go:build e2e

package canton

import (
	"context"
	"encoding/binary"
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

func (s *TestSuite) DeployMCMSWithConfig(config *mcmstypes.Config) {
	s.DeployMCMSContract()

	// Set the config
	configurer, err := cantonsdk.NewConfigurer(s.client, s.participant.UserName, s.participant.Party)
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

	commandID := uuid.Must(uuid.NewUUID()).String()
	cmds := &model.SubmitAndWaitRequest{
		Commands: &model.Commands{
			WorkflowID: "counter-deploy",
			UserID:     s.participant.UserName,
			CommandID:  commandID,
			ActAs:      []string{s.participant.Party},
			Commands:   []*model.Command{{Command: counterContract.CreateCommand()}},
		},
	}

	submitResp, err := s.client.CommandService.SubmitAndWaitForTransaction(s.T().Context(), cmds)
	s.Require().NoError(err)

	// Extract contract ID
	for _, event := range submitResp.Transaction.Events {
		if event.Created != nil && event.Created.TemplateID != "" {
			s.counterCID = event.Created.ContractID
			break
		}
	}
	s.Require().NotEmpty(s.counterCID)
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

// TODO: Remove when validUntil calculation is considered in contracts
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
	// Strip 0x prefix if present
	if strings.HasPrefix(root, "0x") {
		root = root[2:]
	}

	// Inner data: root || validUntil as hex
	validUntilHex := Uint32ToHex(validUntil)
	concatenated := root + validUntilHex
	innerData, err := hex.DecodeString(concatenated)
	if err != nil {
		panic(fmt.Sprintf("Failed to decode hex: %v, concatenated=%s", err, concatenated))
	}
	return crypto.Keccak256Hash(innerData).Bytes()
}

// Uint32ToHex converts uint32 to 32-byte hex (matching Canton's placeholder)
// Canton currently uses padLeft32 "0" - we match that for compatibility
func Uint32ToHex(validUntil uint32) string {
	// For now, match Canton's placeholder implementation
	// TODO: Implement proper timestamp encoding when Canton updates
	return strings.Repeat("0", 64)
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
