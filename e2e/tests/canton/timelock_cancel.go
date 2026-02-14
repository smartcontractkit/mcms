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
	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"
	"github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/stretchr/testify/suite"

	mcmscore "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

type MCMSTimelockCancelTestSuite struct {
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

func TestMCMSTimelockCancelTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSTimelockCancelTestSuite))
}

// SetupSuite runs before the test suite
func (s *MCMSTimelockCancelTestSuite) SetupSuite() {
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

func (s *MCMSTimelockCancelTestSuite) create2of3ProposerConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.proposerAddrs,
	}
}

func (s *MCMSTimelockCancelTestSuite) create2of3CancellerConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.cancellerAddrs,
	}
}

// deployCounterContract deploys a Counter contract for testing
func (s *MCMSTimelockCancelTestSuite) deployCounterContract() {
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

func (s *MCMSTimelockCancelTestSuite) TestTimelockCancel() {
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
		TargetInstanceId: s.counterInstanceID, // Already includes the party from deployCounterContract
		FunctionName:     "Increment",         // Must match Daml choice name (capital I)
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

	// Create timelock proposal with 10 second delay (long enough to cancel)
	delay := mcmstypes.NewDuration(10 * time.Second)
	validUntil := time.Now().Add(24 * time.Hour)

	metadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1, // postOpCount
		s.chainId,
		s.proposerMcmsId,
		s.mcmsId, // MCMS instanceId (without role suffix)
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	timelockProposal, err := mcmscore.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription("Canton Timelock Cancel test - original proposal").
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

	s.T().Logf("SetRoot completed, new MCMS CID: %s", s.mcmsContractID)

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

	s.T().Logf("Schedule operation completed, MCMS CID: %s", s.mcmsContractID)

	// Create timelock inspector to check operation status
	timelockInspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Create timelock executor
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
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().True(isPending, "Operation should be pending after scheduling")

	s.T().Log("Operation is pending, now cancelling...")

	// Create cancellation proposal
	// Note: The canceller role has its own opCount starting at 0, independent of the proposer role
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
	// Note: preOpCount stays at 0 since canceller role hasn't executed any ops yet
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

	// Verify operation is no longer pending (cancelled)
	isPending, err = timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().False(isPending, "Operation should not be pending after cancellation")

	// Verify operation is not done (cancelled, not executed)
	isDone, err := timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationDone should not return an error")
	s.Require().False(isDone, "Operation should not be done after cancellation")

	// Verify operation no longer exists
	isOperation, err := timelockInspector.IsOperation(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperation should not return an error")
	s.Require().False(isOperation, "Operation should not exist after cancellation")

	s.T().Log("TestTimelockCancel completed successfully")
}

// TestTimelockCancel_DeriveFromProposal demonstrates the DeriveCancellationProposal pattern
// similar to Aptos TestTimelock_Cancel, with explicit IsOperation verification before and after.
func (s *MCMSTimelockCancelTestSuite) TestTimelockCancel_DeriveFromProposal() {
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

	// Create timelock proposal with 10 second delay
	delay := mcmstypes.NewDuration(10 * time.Second)
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

	// Step 1: Create the original schedule proposal
	scheduleProposal, err := mcmscore.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription("Canton DeriveCancellationProposal Test - Schedule").
		AddChainMetadata(s.chainSelector, metadata).
		AddTimelockAddress(s.chainSelector, s.mcmsContractID).
		SetAction(mcmstypes.TimelockActionSchedule).
		SetDelay(delay).
		AddOperation(batchOp).
		Build()
	s.Require().NoError(err)

	// Convert to MCMS proposal
	timelockConverter := cantonsdk.NewTimelockConverter()
	convertersMap := map[mcmstypes.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := scheduleProposal.Convert(ctx, convertersMap)
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

	// Sign and execute schedule proposal
	_, _, err = s.SignProposal(&proposal, inspector, s.sortedProposerKeys[:2], 2)
	s.Require().NoError(err)

	executable, err := mcmscore.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	rawData, ok = txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	newMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, s.proposerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	proposal.ChainMetadata[s.chainSelector] = newMetadata

	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)

	rawTxData, ok := txExecute.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawTxData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	s.T().Log("Schedule operation completed")

	// Create timelock inspector
	timelockInspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Create timelock executor
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)
	timelockExecutors := map[mcmstypes.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}

	timelockExecutable, err := mcmscore.NewTimelockExecutable(ctx, scheduleProposal, timelockExecutors)
	s.Require().NoError(err)

	scheduledOpID, err := timelockExecutable.GetOpID(ctx, 0, batchOp, s.chainSelector)
	s.Require().NoError(err)

	// Step 2: Verify IsOperation = true (operation exists)
	isOperation, err := timelockInspector.IsOperation(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isOperation, "IsOperation should return true after scheduling")
	s.T().Log("Verified: IsOperation = true after scheduling")

	// Step 3: Derive cancellation proposal from the original schedule proposal
	// This is the key pattern from Aptos: deriving cancellation from the original proposal
	cancellerMcmsId := fmt.Sprintf("%s-%s", s.mcmsId, "canceller")
	cancelMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, cancellerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)

	// DeriveCancellationProposal creates a new proposal with the same operations but action=Cancel
	cancelTimelockProposal, err := scheduleProposal.DeriveCancellationProposal(map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
		s.chainSelector: cancelMetadata,
	})
	s.Require().NoError(err)
	s.T().Log("Derived cancellation proposal from original schedule proposal")

	// Step 4: Execute the cancellation
	cancelProposal, _, err := cancelTimelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	cancelInspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleCanceller)

	cancelEncoders, err := cancelProposal.GetEncoders()
	s.Require().NoError(err)
	cancelEncoder := cancelEncoders[s.chainSelector].(*cantonsdk.Encoder)

	cancelExecutor, err := cantonsdk.NewExecutor(cancelEncoder, cancelInspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleCanceller)
	s.Require().NoError(err)

	cancelExecutors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: cancelExecutor,
	}

	_, _, err = s.SignProposal(&cancelProposal, cancelInspector, s.sortedCancellerKeys[:2], 2)
	s.Require().NoError(err)

	cancelExecutable, err := mcmscore.NewExecutable(&cancelProposal, cancelExecutors)
	s.Require().NoError(err)

	txCancelSetRoot, err := cancelExecutable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	rawData, ok = txCancelSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok = rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	newCancelMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, cancellerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	cancelProposal.ChainMetadata[s.chainSelector] = newCancelMetadata

	txCancelExecute, err := cancelExecutable.Execute(ctx, 0)
	s.Require().NoError(err)

	rawCancelData, ok := txCancelExecute.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawCancelData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	s.T().Logf("Cancel operation completed in tx: %s", txCancelExecute.Hash)

	// Step 5: Verify IsOperation = false (operation no longer exists)
	isOperation, err = timelockInspector.IsOperation(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isOperation, "IsOperation should return false after cancellation")
	s.T().Log("Verified: IsOperation = false after cancellation")

	// Additional verifications
	isPending, err := timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isPending, "Operation should not be pending after cancellation")

	isDone, err := timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isDone, "Operation should not be done after cancellation")

	s.T().Log("TestTimelockCancel_DeriveFromProposal completed successfully")
}
