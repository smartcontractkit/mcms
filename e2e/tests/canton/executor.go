//go:build e2e

package canton

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

func TestMCMSExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSExecutorTestSuite))
}

// SetupSuite runs before the test suite
func (s *MCMSExecutorTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()

	// Create 3 signers for 2-of-3 multisig
	s.signers = make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < 3; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.signers[i] = key
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

	// Derive sorted addresses from sorted signers to ensure they correspond
	s.signerAddrs = make([]common.Address, len(s.sortedSigners))
	for i, signer := range s.sortedSigners {
		s.sortedWallets[i] = mcmscore.NewPrivateKeySigner(signer)
		s.signerAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}

	// Deploy MCMS with config
	config := s.create2of3Config()
	s.DeployMCMSWithConfig(config)

	// Deploy Counter contract for ExecuteOp tests
	s.DeployCounterContract()
}

func (s *MCMSExecutorTestSuite) create2of3Config() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.signerAddrs,
	}
}

func (s *MCMSExecutorTestSuite) TestSetRootAndExecuteCounterOp() {
	ctx := s.T().Context()

	// Create metadata for Canton chain
	metadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1,
		s.chainId,
		s.proposerMcmsId,
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	// Build a test proposal with an operation to increment counter
	validUntil := time.Now().Add(24 * time.Hour)
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: fmt.Sprintf("%s@%s", s.counterInstanceID, s.participant.Party),
		FunctionName:     "increment",
		OperationData:    "",
		TargetCid:        s.counterCID,
		ContractIds:      []string{s.counterCID},
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	proposal, err := mcmscore.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription(fmt.Sprintf("Canton ExecuteOp test - %v", validUntil)).
		SetOverridePreviousRoot(false).
		AddChainMetadata(s.chainSelector, metadata).
		AddOperation(mcmstypes.Operation{
			ChainSelector: s.chainSelector,
			Transaction: mcmstypes.Transaction{
				To:               s.counterCID,
				Data:             []byte{},
				AdditionalFields: opAdditionalFieldsBytes,
			},
		}).
		Build()
	s.Require().NoError(err)

	// Create inspector and executor
	inspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleProposer)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)

	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)

	executors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign with first 2 sorted signers
	_, _, err = s.SignProposal(proposal, inspector, s.sortedSigners[:2], 2)
	s.Require().NoError(err)

	// Create executable
	executable, err := mcmscore.NewExecutable(proposal, executors)
	s.Require().NoError(err)

	// First, set the root
	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txSetRoot.Hash)

	// Update contract ID after SetRoot
	rawData, ok := txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.Require().NotEmpty(newMCMSContractID)
	s.mcmsContractID = newMCMSContractID

	s.T().Logf("✅ SetRoot completed, new MCMS CID: %s", s.mcmsContractID)

	// Update proposal with new multisig id
	newMetadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1, // postOp
		s.chainId,
		s.proposerMcmsId,
		s.mcmsContractID,
		false,
	)
	proposal.ChainMetadata[s.chainSelector] = newMetadata
	// Now execute the operation (index 0)
	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txExecute.Hash)

	// Verify the counter was incremented by checking transaction events
	rawTx, ok := txExecute.RawData.(map[string]any)["RawTx"]
	s.Require().True(ok)
	submitResp, ok := rawTx.(*apiv2.SubmitAndWaitForTransactionResponse)
	s.Require().True(ok)

	s.verifyCounterIncremented(submitResp)

	// Verify MCMS contract was recreated with incremented opCount
	foundMCMS := false
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalized := cantonsdk.NormalizeTemplateKey(templateID)
			if normalized == cantonsdk.MCMSTemplateKey {
				foundMCMS = true
				newMCMSCID := createdEv.GetContractId()
				s.Require().NotEmpty(newMCMSCID)
				s.Require().NotEqual(s.mcmsContractID, newMCMSCID, "MCMS should be recreated after execute")
				s.mcmsContractID = newMCMSCID
				s.T().Logf("✅ MCMS contract recreated: %s", s.mcmsContractID)
				break
			}
		}
	}
	s.Require().True(foundMCMS, "MCMS contract should be recreated after execute")

	s.T().Logf("✅ ExecuteOp completed in tx: %s", txExecute.Hash)
}

func (s *MCMSExecutorTestSuite) TestSetRootAndExecuteMCMSOp() {
	ctx := s.T().Context()

	// Encode the SetConfig operation data to change quorum from 2 to 1
	// For Canton, we need to encode the signers, group quorums, and group parents
	groupQuorums := make([]int64, 32)
	groupQuorums[0] = 1 // Root group needs 1 signature now
	groupParents := make([]int64, 32)
	// All groups point to root (0)

	// Convert signers to lowercase hex without 0x prefix (Canton format)
	signerAddresses := make([]string, len(s.signerAddrs))
	signerGroups := make([]int64, len(s.signerAddrs))
	for i, addr := range s.signerAddrs {
		signerAddresses[i] = addr.Hex()[2:] // Remove 0x prefix, keep checksum
		signerGroups[i] = 0                 // All in group 0
	}

	// Encode SetConfig params using Canton's encoding format
	// This follows the format from chainlink-canton integration tests
	operationData := EncodeSetConfigParams(&s.TestSuite, signerAddresses, groupQuorums, groupParents, false)

	// Create metadata for Canton chain
	metadata, err := cantonsdk.NewChainMetadata(
		1, // preOpCount
		2, // postOp
		s.chainId,
		s.proposerMcmsId,
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	// Build a proposal with SetConfig operation targeting MCMS itself
	validUntil := time.Now().Add(24 * time.Hour)
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: "self",
		FunctionName:     "set_config",
		OperationData:    operationData,
		TargetCid:        s.mcmsContractID,
		ContractIds:      []string{s.mcmsContractID},
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	proposal, err := mcmscore.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription(fmt.Sprintf("Canton SetConfig test - change quorum to 3 - %v", validUntil)).
		SetOverridePreviousRoot(false).
		AddChainMetadata(s.chainSelector, metadata).
		AddOperation(mcmstypes.Operation{
			ChainSelector: s.chainSelector,
			Transaction: mcmstypes.Transaction{
				To:               s.mcmsContractID,
				Data:             []byte{},
				AdditionalFields: opAdditionalFieldsBytes,
			},
		}).
		Build()
	s.Require().NoError(err)

	// Create inspector and executor
	inspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleProposer)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)

	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)

	executors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign with first 2 sorted signers
	_, _, err = s.SignProposal(proposal, inspector, s.sortedSigners[:2], 2)
	s.Require().NoError(err)

	// Create executable
	executable, err := mcmscore.NewExecutable(proposal, executors)
	s.Require().NoError(err)

	// First, set the root
	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txSetRoot.Hash)

	// Update contract ID after SetRoot
	rawData, ok := txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.Require().NotEmpty(newMCMSContractID)
	oldMCMSContractID := s.mcmsContractID
	s.mcmsContractID = newMCMSContractID
	s.Require().NotEqual(oldMCMSContractID, s.mcmsContractID, "MCMS contract ID should change after SetRoot")

	// Update proposal with new MCMS contract ID in metadata

	s.T().Logf("✅ SetRoot completed, new MCMS CID: %s", s.mcmsContractID)

	newMetadata, err := cantonsdk.NewChainMetadata(
		1, // preOpCount
		2, // postOp
		s.chainId,
		s.proposerMcmsId,
		newMCMSContractID,
		false,
	)
	s.Require().NoError(err)
	// Override proposal metadata with new MCMS contract ID
	proposal.ChainMetadata[s.chainSelector] = newMetadata

	// Now execute the SetConfig operation (index 0)
	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txExecute.Hash)

	// Verify MCMS contract was recreated with new config
	rawTx, ok := txExecute.RawData.(map[string]any)["RawTx"]
	s.Require().True(ok)
	submitResp, ok := rawTx.(*apiv2.SubmitAndWaitForTransactionResponse)
	s.Require().True(ok)

	foundMCMS := false
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalized := cantonsdk.NormalizeTemplateKey(templateID)
			if normalized == cantonsdk.MCMSTemplateKey {
				foundMCMS = true
				newMCMSCID := createdEv.GetContractId()
				s.Require().NotEmpty(newMCMSCID)
				s.Require().NotEqual(oldMCMSContractID, newMCMSCID, "MCMS should be recreated after SetConfig")
				s.mcmsContractID = newMCMSCID
				s.T().Logf("✅ MCMS contract recreated with new config: %s", s.mcmsContractID)
				break
			}
		}
	}
	s.Require().True(foundMCMS, "MCMS contract should be recreated after SetConfig")

	s.T().Logf("✅ SetConfig operation completed in tx: %s", txExecute.Hash)

	// TODO: Inspect config when ready
}

// Helper functions
func (s *MCMSExecutorTestSuite) verifyCounterIncremented(submitResp *apiv2.SubmitAndWaitForTransactionResponse) {
	// Look for Counter contract in created events
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalized := cantonsdk.NormalizeTemplateKey(templateID)
			if normalized == "MCMS.Counter:Counter" {
				// Counter was recreated, which means it was successfully executed
				s.T().Log("Counter contract was successfully incremented")
				return
			}
		}
	}
	s.Fail("Counter contract not found in transaction events")
}
