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
	"github.com/smartcontractkit/go-daml/pkg/service/ledger"
	"github.com/smartcontractkit/go-daml/pkg/types"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/chainlink-canton/bindings/mcms"

	mcmscore "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// MCMSTimelockProposalTestSuite tests combined proposer and bypasser flows
// similar to Aptos TestTimelockProposal
type MCMSTimelockProposalTestSuite struct {
	TestSuite

	// Proposer signers
	proposerSigners    []*ecdsa.PrivateKey
	proposerAddrs      []common.Address
	sortedProposerKeys []*ecdsa.PrivateKey
	proposerWallets    []*mcmscore.PrivateKeySigner

	// Bypasser signers
	bypasserSigners    []*ecdsa.PrivateKey
	bypasserAddrs      []common.Address
	sortedBypasserKeys []*ecdsa.PrivateKey
	bypasserWallets    []*mcmscore.PrivateKeySigner

	// Counter contract for testing
	counterInstanceID string
	counterCID        string
}

func TestMCMSTimelockProposalTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSTimelockProposalTestSuite))
}

// SetupSuite runs before the test suite
func (s *MCMSTimelockProposalTestSuite) SetupSuite() {
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
	s.proposerWallets = make([]*mcmscore.PrivateKeySigner, len(s.sortedProposerKeys))
	s.proposerAddrs = make([]common.Address, len(s.sortedProposerKeys))
	for i, signer := range s.sortedProposerKeys {
		s.proposerWallets[i] = mcmscore.NewPrivateKeySigner(signer)
		s.proposerAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}

	// Create 3 bypasser signers for 2-of-3 multisig
	s.bypasserSigners = make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < 3; i++ {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.bypasserSigners[i] = key
	}

	// Sort bypasser signers by address
	bypassersCopy := make([]*ecdsa.PrivateKey, len(s.bypasserSigners))
	copy(bypassersCopy, s.bypasserSigners)
	slices.SortFunc(bypassersCopy, func(a, b *ecdsa.PrivateKey) int {
		addrA := crypto.PubkeyToAddress(a.PublicKey)
		addrB := crypto.PubkeyToAddress(b.PublicKey)
		return addrA.Cmp(addrB)
	})
	s.sortedBypasserKeys = bypassersCopy
	s.bypasserWallets = make([]*mcmscore.PrivateKeySigner, len(s.sortedBypasserKeys))
	s.bypasserAddrs = make([]common.Address, len(s.sortedBypasserKeys))
	for i, signer := range s.sortedBypasserKeys {
		s.bypasserWallets[i] = mcmscore.NewPrivateKeySigner(signer)
		s.bypasserAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}
}

func (s *MCMSTimelockProposalTestSuite) create2of3ProposerConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.proposerAddrs,
	}
}

func (s *MCMSTimelockProposalTestSuite) create2of3BypasserConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.bypasserAddrs,
	}
}

// deployCounterContract deploys a Counter contract for testing
func (s *MCMSTimelockProposalTestSuite) deployCounterContract() {
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

// TestTimelockProposal tests a combined end-to-end flow:
// Part 1: Proposer path - schedule operation, wait for delay, execute via timelock
// Part 2: Bypasser path - bypass execute with immediate execution
// This mirrors the Aptos TestTimelockProposal pattern.
func (s *MCMSTimelockProposalTestSuite) TestTimelockProposal() {
	ctx := s.T().Context()

	// =====================================================
	// Setup: Deploy MCMS with both proposer and bypasser configs
	// =====================================================
	proposerConfig := s.create2of3ProposerConfig()
	s.DeployMCMSWithConfig(proposerConfig)

	// Set bypasser config
	bypasserConfig := s.create2of3BypasserConfig()
	bypasserConfigurer, err := cantonsdk.NewConfigurer(s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleBypasser)
	s.Require().NoError(err)

	tx, err := bypasserConfigurer.SetConfig(ctx, s.mcmsContractID, bypasserConfig, true)
	s.Require().NoError(err)

	// Update contract ID
	rawData, ok := tx.RawData.(map[string]any)
	s.Require().True(ok)
	newContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newContractID

	s.T().Log("MCMS deployed with proposer and bypasser configs")

	// Deploy Counter contract
	s.deployCounterContract()
	s.T().Logf("Counter deployed with CID: %s", s.counterCID)

	// =====================================================
	// Part 1: PROPOSER PATH - Schedule and Execute
	// =====================================================
	s.T().Log("=== Part 1: Proposer Path (Schedule + Execute) ===")

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

	// Create timelock proposal with 1 second delay
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
		SetDescription("Canton Combined Proposal Test - Proposer Path").
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

	s.T().Logf("Proposer SetRoot completed, new MCMS CID: %s", s.mcmsContractID)

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

	// Wait for delay to pass
	s.T().Log("Waiting for delay to pass...")
	time.Sleep(2 * time.Second)

	// Verify operation is now ready
	isReady, err := timelockInspector.IsOperationReady(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isReady, "Operation should be ready after delay passes")

	// Execute the scheduled batch via TimelockExecutor
	txTimelockExec, err := timelockExecutable.Execute(ctx, 0, mcmscore.WithCallProxy(s.mcmsContractID))
	s.Require().NoError(err)
	s.Require().NotEmpty(txTimelockExec.Hash)

	s.T().Logf("Timelock execute completed in tx: %s", txTimelockExec.Hash)

	// Update MCMS contract ID
	rawExecData, ok := txTimelockExec.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawExecData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	// Verify operation is now done
	isDone, err := timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isDone, "Operation should be done after execution")

	// Verify counter was incremented (value should be 1)
	rawTx, ok := txTimelockExec.RawData.(map[string]any)["RawTx"]
	s.Require().True(ok)
	submitResp, ok := rawTx.(*apiv2.SubmitAndWaitForTransactionResponse)
	s.Require().True(ok)
	s.verifyCounterValue(submitResp, 1)

	s.T().Log("Part 1 completed: Proposer path successful")

	// =====================================================
	// Part 2: BYPASSER PATH - Immediate Execution
	// =====================================================
	s.T().Log("=== Part 2: Bypasser Path (Bypass Execute) ===")

	// Update counter CID from previous execution
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalized := cantonsdk.NormalizeTemplateKey(templateID)
			if normalized == "MCMS.Counter:Counter" {
				s.counterCID = createdEv.GetContractId()
				break
			}
		}
	}

	// Build new batch operation with updated counter CID
	bypassOpAdditionalFields := cantonsdk.AdditionalFields{
		TargetInstanceId: s.counterInstanceID,
		FunctionName:     "Increment",
		OperationData:    "",
		TargetCid:        s.counterCID,
		ContractIds:      []string{s.counterCID},
	}
	bypassOpAdditionalFieldsBytes, err := json.Marshal(bypassOpAdditionalFields)
	s.Require().NoError(err)

	bypassBatchOp := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions: []mcmstypes.Transaction{{
			To:               s.counterCID,
			Data:             []byte("increment"),
			AdditionalFields: bypassOpAdditionalFieldsBytes,
		}},
	}

	// Create bypass timelock proposal (no delay needed)
	bypasserMcmsId := fmt.Sprintf("%s-%s", s.mcmsId, "bypasser")
	bypassMetadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount - bypasser role starts at 0
		1, // postOpCount
		s.chainId,
		bypasserMcmsId,
		s.mcmsId,
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	bypassTimelockProposal, err := mcmscore.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription("Canton Combined Proposal Test - Bypasser Path").
		AddChainMetadata(s.chainSelector, bypassMetadata).
		AddTimelockAddress(s.chainSelector, s.mcmsContractID).
		SetAction(mcmstypes.TimelockActionBypass).
		AddOperation(bypassBatchOp).
		Build()
	s.Require().NoError(err)

	// Convert timelock proposal to MCMS proposal
	bypassProposal, _, err := bypassTimelockProposal.Convert(ctx, convertersMap)
	s.Require().NoError(err)

	// Create inspector and executor for bypasser role
	bypassInspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleBypasser)

	bypassEncoders, err := bypassProposal.GetEncoders()
	s.Require().NoError(err)
	bypassEncoder := bypassEncoders[s.chainSelector].(*cantonsdk.Encoder)

	bypassExecutor, err := cantonsdk.NewExecutor(bypassEncoder, bypassInspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleBypasser)
	s.Require().NoError(err)

	bypassExecutors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: bypassExecutor,
	}

	// Sign with first 2 sorted bypasser signers
	_, _, err = s.SignProposal(&bypassProposal, bypassInspector, s.sortedBypasserKeys[:2], 2)
	s.Require().NoError(err)

	// Create executable
	bypassExecutable, err := mcmscore.NewExecutable(&bypassProposal, bypassExecutors)
	s.Require().NoError(err)

	// Set the root
	txBypassSetRoot, err := bypassExecutable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)
	s.Require().NotEmpty(txBypassSetRoot.Hash)

	// Update contract ID after SetRoot
	rawData, ok = txBypassSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok = rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	s.T().Logf("Bypasser SetRoot completed, new MCMS CID: %s", s.mcmsContractID)

	// Update proposal metadata with new contract ID
	newBypassMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, bypasserMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	bypassProposal.ChainMetadata[s.chainSelector] = newBypassMetadata

	// Execute the bypass operation (immediate execution, no scheduling)
	txBypassExecute, err := bypassExecutable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txBypassExecute.Hash)

	s.T().Logf("Bypass execute completed in tx: %s", txBypassExecute.Hash)

	// Verify counter was incremented (value should be 2 now)
	rawBypassTx, ok := txBypassExecute.RawData.(map[string]any)["RawTx"]
	s.Require().True(ok)
	bypassSubmitResp, ok := rawBypassTx.(*apiv2.SubmitAndWaitForTransactionResponse)
	s.Require().True(ok)
	s.verifyCounterValue(bypassSubmitResp, 2)

	s.T().Log("Part 2 completed: Bypasser path successful")
	s.T().Log("TestTimelockProposal completed successfully - both proposer and bypasser paths verified")
}

// verifyCounterValue checks that the Counter contract has the expected value
func (s *MCMSTimelockProposalTestSuite) verifyCounterValue(submitResp *apiv2.SubmitAndWaitForTransactionResponse, expectedValue int64) {
	// Look for Counter contract in created events
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalized := cantonsdk.NormalizeTemplateKey(templateID)
			if normalized == "MCMS.Counter:Counter" {
				// Counter was recreated, which means it was successfully executed
				// Check the value field
				args := createdEv.GetCreateArguments()
				if args != nil {
					for _, field := range args.GetFields() {
						if field.GetLabel() == "value" {
							actualValue := field.GetValue().GetInt64()
							s.Require().Equal(expectedValue, actualValue, "Counter value should be %d", expectedValue)
							s.T().Logf("Counter value verified: %d", actualValue)
							return
						}
					}
				}
				s.T().Log("Counter contract was successfully incremented")
				return
			}
		}
	}
	s.Fail("Counter contract not found in transaction events")
}
