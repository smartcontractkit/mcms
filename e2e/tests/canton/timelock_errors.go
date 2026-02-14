//go:build e2e

package canton

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"
	"github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"

	mcmscore "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// MCMSTimelockErrorTestSuite tests error cases for timelock operations
type MCMSTimelockErrorTestSuite struct {
	TestSuite

	// Proposer signers
	proposerSigners    []*ecdsa.PrivateKey
	proposerAddrs      []common.Address
	sortedProposerKeys []*ecdsa.PrivateKey

	// Canceller signers
	cancellerSigners    []*ecdsa.PrivateKey
	cancellerAddrs      []common.Address
	sortedCancellerKeys []*ecdsa.PrivateKey

	// Counter contract for testing
	counterInstanceID string
	counterCID        string
}

func TestMCMSTimelockErrorTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSTimelockErrorTestSuite))
}

// SetupSuite runs before the test suite
func (s *MCMSTimelockErrorTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()

	// Create 3 proposer signers for 2-of-3 multisig
	s.proposerSigners = make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < 3; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.proposerSigners[i] = key
	}

	// Sort proposer signers by address
	signersCopy := make([]*ecdsa.PrivateKey, len(s.proposerSigners))
	copy(signersCopy, s.proposerSigners)
	slices.SortFunc(signersCopy, func(a, b *ecdsa.PrivateKey) int {
		addrA := crypto.PubkeyToAddress(a.PublicKey)
		addrB := crypto.PubkeyToAddress(b.PublicKey)
		return addrA.Cmp(addrB)
	})
	s.sortedProposerKeys = signersCopy
	s.proposerAddrs = make([]common.Address, len(s.sortedProposerKeys))
	for i, signer := range s.sortedProposerKeys {
		s.proposerAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}

	// Create 3 canceller signers for 2-of-3 multisig
	s.cancellerSigners = make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < 3; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.cancellerSigners[i] = key
	}

	// Sort canceller signers by address
	cancellersCopy := make([]*ecdsa.PrivateKey, len(s.cancellerSigners))
	copy(cancellersCopy, s.cancellerSigners)
	slices.SortFunc(cancellersCopy, func(a, b *ecdsa.PrivateKey) int {
		addrA := crypto.PubkeyToAddress(a.PublicKey)
		addrB := crypto.PubkeyToAddress(b.PublicKey)
		return addrA.Cmp(addrB)
	})
	s.sortedCancellerKeys = cancellersCopy
	s.cancellerAddrs = make([]common.Address, len(s.sortedCancellerKeys))
	for i, signer := range s.sortedCancellerKeys {
		s.cancellerAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}
}

func (s *MCMSTimelockErrorTestSuite) create2of3ProposerConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.proposerAddrs,
	}
}

func (s *MCMSTimelockErrorTestSuite) create2of3CancellerConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.cancellerAddrs,
	}
}

// deployCounterContract deploys a Counter contract for testing
func (s *MCMSTimelockErrorTestSuite) deployCounterContract() {
	// Instance ID must include the party (Canton convention)
	baseInstanceID := "counter-" + uuid.New().String()[:8]
	s.counterInstanceID = fmt.Sprintf("%s@%s", baseInstanceID, s.participant.Party)

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

// TestTimelockError_MinDelayNotPassed tests that executing before min delay fails
func (s *MCMSTimelockErrorTestSuite) TestTimelockError_MinDelayNotPassed() {
	ctx := s.T().Context()

	// Deploy MCMS with proposer config
	proposerConfig := s.create2of3ProposerConfig()
	s.DeployMCMSWithConfig(proposerConfig)

	// Deploy Counter contract
	s.deployCounterContract()

	// Build batch operation to increment counter
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: s.counterInstanceID,
		FunctionName:     "Increment",
		OperationData:    "",
		TargetCid:        s.counterCID,
		ContractIds:      []string{s.counterCID},
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	batchOp := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions: []mcmstypes.Transaction{{
			To:               s.counterCID,
			Data:             []byte("increment"),
			AdditionalFields: opAdditionalFieldsBytes,
		}},
	}

	// Create timelock proposal with 60 second delay (long enough to test immediate execution failure)
	delay := mcmstypes.NewDuration(60 * time.Second)
	validUntil := time.Now().Add(24 * time.Hour)

	metadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1, // postOpCount
		s.chainId,
		s.proposerMcmsId,
		s.mcmsId,
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	timelockProposal, err := mcmscore.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription("Canton Error Test - Min Delay Not Passed").
		AddChainMetadata(s.chainSelector, metadata).
		AddTimelockAddress(s.chainSelector, s.mcmsContractID).
		SetAction(mcmstypes.TimelockActionSchedule).
		SetDelay(delay).
		AddOperation(batchOp).
		Build()
	s.Require().NoError(err)

	// Convert timelock proposal to MCMS proposal
	timelockConverter := cantonsdk.NewTimelockConverter()
	convertersMap := map[mcmstypes.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	// Create inspector and executor for proposer role
	inspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleProposer)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)

	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)

	executors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign with first 2 sorted proposer signers
	_, _, err = s.SignProposal(&proposal, inspector, s.sortedProposerKeys[:2], 2)
	s.Require().NoError(err)

	// Create executable
	executable, err := mcmscore.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// Set the root
	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txSetRoot.Hash)

	// Update contract ID after SetRoot
	rawData, ok := txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	// Update proposal metadata with new contract ID
	newMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, s.proposerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	proposal.ChainMetadata[s.chainSelector] = newMetadata

	// Execute the operation (this schedules the batch)
	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txExecute.Hash)

	// Update MCMS contract ID from execute result
	rawTxData, ok := txExecute.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawTxData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	s.T().Log("Operation scheduled, now attempting immediate execution (should fail)")

	// Create timelock inspector to check operation status
	timelockInspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Create timelock executor for executing the scheduled batch
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)
	timelockExecutors := map[mcmstypes.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}

	// Create timelock executable
	timelockExecutable, err := mcmscore.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	// Get the operation ID
	scheduledOpID, err := timelockExecutable.GetOpID(ctx, 0, batchOp, s.chainSelector)
	s.Require().NoError(err, "Failed to get operation ID")

	// Verify operation is pending but not ready
	isPending, err := timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isPending, "Operation should be pending after scheduling")

	isReady, err := timelockInspector.IsOperationReady(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isReady, "Operation should NOT be ready before delay passes")

	// Attempt to execute the scheduled batch immediately (should fail)
	_, err = timelockExecutable.Execute(ctx, 0, mcmscore.WithCallProxy(s.mcmsContractID))
	s.Require().Error(err, "Execute should fail when min delay has not passed")

	// Verify the error message indicates the operation is not ready
	s.T().Logf("Expected error received: %v", err)
	s.Require().True(
		strings.Contains(err.Error(), "E_NOT_READY"),
		"Error should indicate operation is not ready for execution",
	)

	s.T().Log("TestTimelockError_MinDelayNotPassed completed successfully - execution correctly rejected")
}

// TestTimelockError_ExecuteAfterCancel tests that executing a cancelled operation fails
func (s *MCMSTimelockErrorTestSuite) TestTimelockError_ExecuteAfterCancel() {
	ctx := s.T().Context()

	// Deploy MCMS with proposer config
	proposerConfig := s.create2of3ProposerConfig()
	s.DeployMCMSWithConfig(proposerConfig)

	// Also set canceller config
	cancellerConfig := s.create2of3CancellerConfig()
	configurer, err := cantonsdk.NewConfigurer(s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleCanceller)
	s.Require().NoError(err)

	tx, err := configurer.SetConfig(ctx, s.mcmsContractID, cancellerConfig, true)
	s.Require().NoError(err)

	// Update contract ID
	rawData, ok := tx.RawData.(map[string]any)
	s.Require().True(ok)
	newContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newContractID

	// Deploy Counter contract
	s.deployCounterContract()

	// Build batch operation to increment counter
	opAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: s.counterInstanceID,
		FunctionName:     "Increment",
		OperationData:    "",
		TargetCid:        s.counterCID,
		ContractIds:      []string{s.counterCID},
	}
	opAdditionalFieldsBytes, err := json.Marshal(opAdditionalFields)
	s.Require().NoError(err)

	batchOp := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions: []mcmstypes.Transaction{{
			To:               s.counterCID,
			Data:             []byte("increment"),
			AdditionalFields: opAdditionalFieldsBytes,
		}},
	}

	// Create timelock proposal with 1 second delay (short delay so we can test execution after cancel)
	delay := mcmstypes.NewDuration(1 * time.Second)
	validUntil := time.Now().Add(24 * time.Hour)

	metadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1, // postOpCount
		s.chainId,
		s.proposerMcmsId,
		s.mcmsId,
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	timelockProposal, err := mcmscore.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription("Canton Error Test - Execute After Cancel").
		AddChainMetadata(s.chainSelector, metadata).
		AddTimelockAddress(s.chainSelector, s.mcmsContractID).
		SetAction(mcmstypes.TimelockActionSchedule).
		SetDelay(delay).
		AddOperation(batchOp).
		Build()
	s.Require().NoError(err)

	// Convert timelock proposal to MCMS proposal
	timelockConverter := cantonsdk.NewTimelockConverter()
	convertersMap := map[mcmstypes.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	// Create inspector and executor for proposer role
	inspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleProposer)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)

	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleProposer)
	s.Require().NoError(err)

	executors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign with first 2 sorted proposer signers
	_, _, err = s.SignProposal(&proposal, inspector, s.sortedProposerKeys[:2], 2)
	s.Require().NoError(err)

	// Create executable
	executable, err := mcmscore.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// Set the root
	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txSetRoot.Hash)

	// Update contract ID after SetRoot
	rawData, ok = txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	// Update proposal metadata with new contract ID
	newMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, s.proposerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	proposal.ChainMetadata[s.chainSelector] = newMetadata

	// Execute the operation (this schedules the batch)
	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txExecute.Hash)

	// Update MCMS contract ID from execute result
	rawTxData, ok := txExecute.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawTxData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	s.T().Log("Operation scheduled, now cancelling...")

	// Create timelock inspector
	timelockInspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Create timelock executor for executing the scheduled batch
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)
	timelockExecutors := map[mcmstypes.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}

	// Create timelock executable
	timelockExecutable, err := mcmscore.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	// Get the operation ID
	scheduledOpID, err := timelockExecutable.GetOpID(ctx, 0, batchOp, s.chainSelector)
	s.Require().NoError(err, "Failed to get operation ID")

	// Verify operation is pending (scheduled)
	isPending, err := timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isPending, "Operation should be pending after scheduling")

	// =====================================================
	// Cancel the operation
	// =====================================================
	cancellerMcmsId := fmt.Sprintf("%s-%s", s.mcmsId, "canceller")
	cancelMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, cancellerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)

	cancelTimelockProposal, err := timelockProposal.DeriveCancellationProposal(map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
		s.chainSelector: cancelMetadata,
	})
	s.Require().NoError(err)

	// Convert cancel proposal to MCMS proposal
	cancelProposal, _, err := cancelTimelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	// Create inspector and executor for canceller role
	cancelInspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleCanceller)

	cancelEncoders, err := cancelProposal.GetEncoders()
	s.Require().NoError(err)
	cancelEncoder := cancelEncoders[s.chainSelector].(*cantonsdk.Encoder)

	cancelExecutor, err := cantonsdk.NewExecutor(cancelEncoder, cancelInspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleCanceller)
	s.Require().NoError(err)

	cancelExecutors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: cancelExecutor,
	}

	// Sign cancellation proposal with canceller signers
	_, _, err = s.SignProposal(&cancelProposal, cancelInspector, s.sortedCancellerKeys[:2], 2)
	s.Require().NoError(err)

	// Create cancel executable
	cancelExecutable, err := mcmscore.NewExecutable(&cancelProposal, cancelExecutors)
	s.Require().NoError(err)

	// Set root for cancellation proposal
	txCancelSetRoot, err := cancelExecutable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txCancelSetRoot.Hash)

	// Update contract ID
	rawData, ok = txCancelSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok = rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	// Update cancel proposal metadata
	newCancelMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, cancellerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	cancelProposal.ChainMetadata[s.chainSelector] = newCancelMetadata

	// Execute the cancellation
	txCancelExecute, err := cancelExecutable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txCancelExecute.Hash)

	s.T().Logf("Cancel operation completed in tx: %s", txCancelExecute.Hash)

	// Update MCMS contract ID
	rawCancelData, ok := txCancelExecute.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawCancelData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	// Verify operation no longer exists
	isOperation, err := timelockInspector.IsOperation(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isOperation, "Operation should not exist after cancellation")

	// =====================================================
	// Attempt to execute the cancelled operation (should fail)
	// =====================================================
	s.T().Log("Attempting to execute cancelled operation (should fail)")

	// Wait for delay to pass to ensure the error is about cancellation, not timing
	time.Sleep(2 * time.Second)

	// Attempt to execute the cancelled operation
	_, err = timelockExecutable.Execute(ctx, 0, mcmscore.WithCallProxy(s.mcmsContractID))
	s.Require().Error(err, "Execute should fail for cancelled operation")

	// Verify the error message indicates the operation doesn't exist or was cancelled
	s.T().Logf("Expected error received: %v", err)
	s.Require().True(
		strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "NOT_FOUND") ||
			strings.Contains(err.Error(), "cancelled") ||
			strings.Contains(err.Error(), "does not exist") ||
			strings.Contains(err.Error(), "unknown") ||
			strings.Contains(err.Error(), "UNKNOWN"),
		"Error should indicate operation not found or cancelled",
	)

	s.T().Log("TestTimelockError_ExecuteAfterCancel completed successfully - execution correctly rejected")
}
