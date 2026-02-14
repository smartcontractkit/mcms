//go:build e2e

package canton

import (
	"context"
	"io"
	"slices"
	"testing"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

type MCMSInspectorTestSuite struct {
	TestSuite
	inspector *cantonsdk.Inspector
}

// SetupSuite runs before the test suite
func (s *MCMSInspectorTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()
	s.DeployMCMSContract()

	// Create inspector instance using participant's StateServiceClient
	s.inspector = cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleProposer)
}

func (s *MCMSInspectorTestSuite) TestGetConfig() {
	ctx := s.T().Context()

	// Signers in each group need to be sorted alphabetically
	signers := [30]common.Address{}
	for i := range signers {
		key, _ := crypto.GenerateKey()
		signers[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	slices.SortFunc(signers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	expectedConfig := &mcmstypes.Config{
		Quorum: 2,
		Signers: []common.Address{
			signers[0],
			signers[1],
			signers[2],
		},
		GroupSigners: []mcmstypes.Config{
			{
				Quorum: 4,
				Signers: []common.Address{
					signers[3],
					signers[4],
					signers[5],
					signers[6],
					signers[7],
				},
				GroupSigners: []mcmstypes.Config{
					{
						Quorum: 1,
						Signers: []common.Address{
							signers[8],
							signers[9],
						},
						GroupSigners: []mcmstypes.Config{},
					},
				},
			},
			{
				Quorum: 3,
				Signers: []common.Address{
					signers[10],
					signers[11],
					signers[12],
					signers[13],
				},
				GroupSigners: []mcmstypes.Config{},
			},
		},
	}

	// Set config using configurer
	configurer, err := cantonsdk.NewConfigurer(s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err, "creating configurer")

	tx, err := configurer.SetConfig(ctx, s.mcmsContractID, expectedConfig, true)
	s.Require().NoError(err, "setting config")

	// Get the new contract ID from the SetConfig result (not from getLatestMCMSContractID
	// which may return a different contract if multiple MCMS contracts exist)
	rawData, ok := tx.RawData.(map[string]any)
	s.Require().True(ok, "tx.RawData should be map[string]any")
	newContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok, "NewMCMSContractID should be a string")
	s.Require().NotEmpty(newContractID, "NewMCMSContractID should not be empty")

	// Now test the inspector
	actualConfig, err := s.inspector.GetConfig(ctx, newContractID)
	s.Require().NoError(err, "getting config from inspector")
	s.Require().NotNil(actualConfig, "config should not be nil")

	// Verify the config matches what we set
	s.verifyConfigMatch(expectedConfig, actualConfig)
}

func (s *MCMSInspectorTestSuite) TestGetOpCount() {
	ctx := s.T().Context()

	// Get the latest contract ID (may have changed if other tests ran first)
	contractID, err := s.getLatestMCMSContractID(ctx)
	s.Require().NoError(err, "getting latest MCMS contract ID")

	opCount, err := s.inspector.GetOpCount(ctx, contractID)
	s.Require().NoError(err, "getting op count")

	// Op count may be non-zero if other tests ran first
	s.T().Logf("Current op count: %d", opCount)
}

func (s *MCMSInspectorTestSuite) TestGetRoot() {
	ctx := s.T().Context()

	// Get the latest contract ID (may have changed if other tests ran first)
	contractID, err := s.getLatestMCMSContractID(ctx)
	s.Require().NoError(err, "getting latest MCMS contract ID")

	root, validUntil, err := s.inspector.GetRoot(ctx, contractID)
	s.Require().NoError(err, "getting root")

	// Log values - they may be non-empty if other tests ran first
	s.T().Logf("Root: %s, validUntil: %d (0x%x)", root.Hex(), validUntil, validUntil)
}

func (s *MCMSInspectorTestSuite) TestGetRootMetadata() {
	ctx := s.T().Context()

	// Get the latest contract ID (may have changed if other tests ran first)
	contractID, err := s.getLatestMCMSContractID(ctx)
	s.Require().NoError(err, "getting latest MCMS contract ID")

	metadata, err := s.inspector.GetRootMetadata(ctx, contractID)
	s.Require().NoError(err, "getting root metadata")

	// Verify metadata structure
	s.Require().NotEmpty(metadata.MCMAddress, "MCM address should not be empty")
	s.T().Logf("Metadata - StartingOpCount: %d, MCMAddress: %s", metadata.StartingOpCount, metadata.MCMAddress)
}

// Helper function to get the latest MCMS contract ID
func (s *MCMSInspectorTestSuite) getLatestMCMSContractID(ctx context.Context) (string, error) {
	// Get current ledger offset
	ledgerEndResp, err := s.participant.StateServiceClient.GetLedgerEnd(ctx, &apiv2.GetLedgerEndRequest{})
	if err != nil {
		return "", err
	}

	// Query active contracts
	activeContractsResp, err := s.participant.StateServiceClient.GetActiveContracts(ctx, &apiv2.GetActiveContractsRequest{
		ActiveAtOffset: ledgerEndResp.GetOffset(),
		EventFormat: &apiv2.EventFormat{
			FiltersByParty: map[string]*apiv2.Filters{
				s.participant.Party: {
					Cumulative: []*apiv2.CumulativeFilter{
						{
							IdentifierFilter: &apiv2.CumulativeFilter_TemplateFilter{
								TemplateFilter: &apiv2.TemplateFilter{
									TemplateId: &apiv2.Identifier{
										PackageId:  "#mcms",
										ModuleName: "MCMS.Main",
										EntityName: "MCMS",
									},
									IncludeCreatedEventBlob: false,
								},
							},
						},
					},
				},
			},
			Verbose: true,
		},
	})
	if err != nil {
		return "", err
	}
	defer activeContractsResp.CloseSend()

	// Get the first (and should be only) active MCMS contract
	for {
		resp, err := activeContractsResp.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		activeContract, ok := resp.GetContractEntry().(*apiv2.GetActiveContractsResponse_ActiveContract)
		if !ok {
			continue
		}

		createdEvent := activeContract.ActiveContract.GetCreatedEvent()
		if createdEvent == nil {
			continue
		}

		return createdEvent.ContractId, nil
	}

	return "", nil
}

// Helper to verify config matches
func (s *MCMSInspectorTestSuite) verifyConfigMatch(expected, actual *mcmstypes.Config) {
	s.Require().Equal(expected.Quorum, actual.Quorum, "quorum should match")
	s.Require().Equal(len(expected.Signers), len(actual.Signers), "number of signers should match")

	// Verify signers
	for i, expectedSigner := range expected.Signers {
		s.Require().Equal(expectedSigner, actual.Signers[i], "signer %d should match", i)
	}

	// Verify group signers recursively
	s.Require().Equal(len(expected.GroupSigners), len(actual.GroupSigners), "number of group signers should match")
	for i, expectedGroup := range expected.GroupSigners {
		s.verifyConfigMatch(&expectedGroup, &actual.GroupSigners[i])
	}
}

func TestMCMSInspectorSuite(t *testing.T) {
	suite.Run(t, new(MCMSInspectorTestSuite))
}
