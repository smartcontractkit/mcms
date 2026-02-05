//go:build e2e

package canton

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/noders-team/go-daml/pkg/model"
	"github.com/noders-team/go-daml/pkg/types"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/stretchr/testify/suite"

	mcmscore "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

type MCMSExecutorTestSuite struct {
	TestSuite

	// Test signers
	signers       []*ecdsa.PrivateKey
	signerAddrs   []common.Address
	sortedSigners []*ecdsa.PrivateKey
	sortedWallets []*mcmscore.PrivateKeySigner

	// Counter contract for testing ExecuteOp
	counterInstanceID string
	counterCID        string
}

// SetupSuite runs before the test suite
func (s *MCMSExecutorTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()

	// Create 3 signers for 2-of-3 multisig
	s.signers = make([]*ecdsa.PrivateKey, 3)
	s.signerAddrs = make([]common.Address, 3)
	for i := 0; i < 3; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.signers[i] = key
		s.signerAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
	}

	// Sort signers by address
	signersCopy := make([]*ecdsa.PrivateKey, len(s.signers))
	copy(signersCopy, s.signers)
	slices.SortFunc(signersCopy, func(a, b *ecdsa.PrivateKey) int {
		addrA := crypto.PubkeyToAddress(a.PublicKey)
		addrB := crypto.PubkeyToAddress(b.PublicKey)
		return addrA.Cmp(addrB)
	})
	s.sortedSigners = signersCopy
	s.sortedWallets = make([]*mcmscore.PrivateKeySigner, len(s.sortedSigners))
	for i, signer := range s.sortedSigners {
		s.sortedWallets[i] = mcmscore.NewPrivateKeySigner(signer)
	}

	// Sort addresses as well
	slices.SortFunc(s.signerAddrs, func(a, b common.Address) int {
		return a.Cmp(b)
	})

	// Deploy MCMS with config
	config := s.create2of3Config()
	s.deployMCMSWithConfig(config)

	// Deploy Counter contract for ExecuteOp tests
	s.deployCounterContract()
}

func (s *MCMSExecutorTestSuite) create2of3Config() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.signerAddrs,
	}
}

// TODO: Move to common
func (s *MCMSExecutorTestSuite) deployMCMSWithConfig(config *mcmstypes.Config) {
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

// TODO: Move to common
func (s *MCMSExecutorTestSuite) deployCounterContract() {
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

func (s *MCMSExecutorTestSuite) TestSetRoot() {
	ctx := s.T().Context()

	// Create metadata for Canton chain
	metadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1, // postOpCount
		s.chainId,
		s.mcmsId,
		s.mcmsContractID,
		false, // overridePreviousRoot
	)
	s.Require().NoError(err)

	// Build a test proposal
	validUntil := time.Now().Add(24 * time.Hour)
	proposal, err := mcmscore.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription(fmt.Sprintf("Canton SetRoot test - %v", validUntil)).
		SetOverridePreviousRoot(false).
		AddChainMetadata(s.chainSelector, metadata).
		AddOperation(mcmstypes.Operation{
			ChainSelector: s.chainSelector,
			Transaction: mcmstypes.Transaction{
				To:   s.counterCID,
				Data: []byte{},
				AdditionalFields: json.RawMessage(fmt.Sprintf(`{
					"targetInstanceId": "%s@%s",
					"functionName": "increment",
					"operationData": "",
					"targetCid": "%s",
					"contractIds": ["%s"]
				}`, s.counterInstanceID, s.participant.Party, s.counterCID, s.counterCID)),
			},
		}).
		Build()
	s.Require().NoError(err)

	// Create inspector and executor
	inspector, err := cantonsdk.NewInspector()
	s.Require().NoError(err)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)

	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.client, s.participant.UserName, s.participant.Party)
	s.Require().NoError(err)

	executors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign with first 2 sorted signers
	for i := 0; i < 2; i++ {
		_ = s.SignProposal(proposal, *s.sortedWallets[i])
		s.Require().NoError(err)
	}

	// Create executable and call SetRoot
	executable, err := mcmscore.NewExecutable(proposal, executors)
	s.Require().NoError(err)

	tx, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	// Verify transaction result
	s.Require().NotEmpty(tx.Hash)
	s.Require().Equal("canton", tx.ChainFamily)

	rawData, ok := tx.RawData.(map[string]any)
	s.Require().True(ok)

	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.Require().NotEmpty(newMCMSContractID)
	s.Require().NotEqual(s.mcmsContractID, newMCMSContractID, "new contract ID should be different")

	// Update contract ID for subsequent tests
	s.mcmsContractID = newMCMSContractID

	s.T().Logf("✅ SetRoot in tx: %s", tx.Hash)
}

// Helper functions

func (s *MCMSExecutorTestSuite) verifyCounterIncremented(submitResp *model.SubmitAndWaitForTransactionResponse) {
	// Look for Counter contract in created events
	for _, event := range submitResp.Transaction.Events {
		if event.Created != nil {
			normalized := cantonsdk.NormalizeTemplateKey(event.Created.TemplateID)
			if normalized == "MCMS.Counter:Counter" {
				// Counter was recreated, which means it was successfully executed
				s.T().Log("Counter contract was successfully incremented")
				return
			}
		}
	}
	s.Fail("Counter contract not found in transaction events")
}

func TestMCMSExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSExecutorTestSuite))
}
