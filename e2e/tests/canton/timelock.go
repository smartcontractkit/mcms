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

type MCMSTimelockTestSuite struct {
	TestSuite

	// Test signers for different roles
	proposerSigners    []*ecdsa.PrivateKey
	proposerAddrs      []common.Address
	sortedProposerKeys []*ecdsa.PrivateKey
	proposerWallets    []*mcmscore.PrivateKeySigner

	bypasserSigners    []*ecdsa.PrivateKey
	bypasserAddrs      []common.Address
	sortedBypasserKeys []*ecdsa.PrivateKey
	bypasserWallets    []*mcmscore.PrivateKeySigner

	// Counter contract for testing
	counterInstanceID string
	counterCID        string
}

func TestMCMSTimelockTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSTimelockTestSuite))
}

// SetupSuite runs before the test suite
func (s *MCMSTimelockTestSuite) SetupSuite() {
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

func (s *MCMSTimelockTestSuite) create2of3ProposerConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.proposerAddrs,
	}
}

func (s *MCMSTimelockTestSuite) create2of3BypasserConfig() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.bypasserAddrs,
	}
}

// deployCounterContract deploys a Counter contract for testing
func (s *MCMSTimelockTestSuite) deployCounterContract() {
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

func (s *MCMSTimelockTestSuite) TestTimelockScheduleAndExecute() {
	ctx := s.T().Context()

	// Deploy MCMS with proposer config
	proposerConfig := s.create2of3ProposerConfig()
	s.DeployMCMSWithConfig(proposerConfig)

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

	// Create timelock proposal with 1 second delay
	delay := mcmstypes.NewDuration(1 * time.Second)
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
		SetDescription("Canton Timelock Schedule test").
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
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().True(isPending, "Operation should be pending after scheduling")

	// Verify operation is not ready yet (delay hasn't passed)
	isReady, err := timelockInspector.IsOperationReady(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationReady should not return an error")
	// Note: With 1 second delay and Canton's execution time, it might already be ready

	// Verify operation is not done yet
	isDone, err := timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationDone should not return an error")
	s.Require().False(isDone, "Operation should not be done before execution")

	// Wait for delay to pass
	s.T().Log("Waiting for delay to pass...")
	time.Sleep(2 * time.Second)

	// Verify operation is now ready
	isReady, err = timelockInspector.IsOperationReady(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationReady should not return an error")
	s.Require().True(isReady, "Operation should be ready after delay passes")

	// Execute the scheduled batch via TimelockExecutor
	txTimelockExec, err := timelockExecutable.Execute(ctx, 0, mcmscore.WithCallProxy(s.mcmsContractID))
	s.Require().NoError(err, "Failed to execute scheduled batch")
	s.Require().NotEmpty(txTimelockExec.Hash)

	s.T().Logf("Timelock execute completed in tx: %s", txTimelockExec.Hash)

	// Update MCMS contract ID
	rawExecData, ok := txTimelockExec.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawExecData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	// Verify operation is now done
	isDone, err = timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationDone should not return an error")
	s.Require().True(isDone, "Operation should be done after execution")

	// Verify operation is no longer pending
	isPending, err = timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().False(isPending, "Operation should not be pending after execution")

	// Verify counter was incremented
	rawTx, ok := txTimelockExec.RawData.(map[string]any)["RawTx"]
	s.Require().True(ok)
	submitResp, ok := rawTx.(*apiv2.SubmitAndWaitForTransactionResponse)
	s.Require().True(ok)
	s.verifyCounterValue(submitResp, 1)

	s.T().Log("TestTimelockScheduleAndExecute completed successfully")
}

func (s *MCMSTimelockTestSuite) TestTimelockBypass() {
	ctx := s.T().Context()

	// Deploy MCMS with bypasser config
	bypasserConfig := s.create2of3BypasserConfig()
	s.DeployMCMSContract()

	// Set bypasser config
	configurer, err := cantonsdk.NewConfigurer(s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleBypasser)
	s.Require().NoError(err)

	tx, err := configurer.SetConfig(ctx, s.mcmsContractID, bypasserConfig, true)
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

	// Create bypass timelock proposal (no delay needed)
	validUntil := time.Now().Add(24 * time.Hour)

	bypasserMcmsId := fmt.Sprintf("%s-%s", s.mcmsId, "bypasser")
	metadata, err := cantonsdk.NewChainMetadata(
		0, // preOpCount
		1, // postOpCount
		s.chainId,
		bypasserMcmsId,
		s.mcmsId, // MCMS instanceId (without role suffix)
		s.mcmsContractID,
		false,
	)
	s.Require().NoError(err)

	timelockProposal, err := mcmscore.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(validUntil.Unix())).
		SetDescription("Canton Timelock Bypass test").
		AddChainMetadata(s.chainSelector, metadata).
		AddTimelockAddress(s.chainSelector, s.mcmsContractID).
		SetAction(mcmstypes.TimelockActionBypass).
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

	// Create inspector and executor for bypasser role
	inspector := cantonsdk.NewInspector(s.participant.StateServiceClient, s.participant.Party, cantonsdk.TimelockRoleBypasser)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.chainSelector].(*cantonsdk.Encoder)

	executor, err := cantonsdk.NewExecutor(encoder, inspector, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party, cantonsdk.TimelockRoleBypasser)
	s.Require().NoError(err)

	executors := map[mcmstypes.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}

	// Sign with first 2 sorted bypasser signers
	_, _, err = s.SignProposal(&proposal, inspector, s.sortedBypasserKeys[:2], 2)
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
	newMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, bypasserMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	proposal.ChainMetadata[s.chainSelector] = newMetadata

	// Execute the bypass operation (immediate execution, no scheduling)
	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)
	s.Require().NotEmpty(txExecute.Hash)

	s.T().Logf("Bypass execute completed in tx: %s", txExecute.Hash)

	// Verify counter was incremented
	rawTx, ok := txExecute.RawData.(map[string]any)["RawTx"]
	s.Require().True(ok)
	submitResp, ok := rawTx.(*apiv2.SubmitAndWaitForTransactionResponse)
	s.Require().True(ok)
	s.verifyCounterValue(submitResp, 1)

	s.T().Log("TestTimelockBypass completed successfully")
}

// verifyCounterValue checks that the Counter contract has the expected value
func (s *MCMSTimelockTestSuite) verifyCounterValue(submitResp *apiv2.SubmitAndWaitForTransactionResponse, expectedValue int64) {
	transaction := submitResp.GetTransaction()
	for _, event := range transaction.GetEvents() {
		if createdEv := event.GetCreated(); createdEv != nil {
			templateID := cantonsdk.FormatTemplateID(createdEv.GetTemplateId())
			normalized := cantonsdk.NormalizeTemplateKey(templateID)
			if normalized == "MCMS.Counter:Counter" {
				// Extract and verify the counter value from create arguments
				args := createdEv.GetCreateArguments()
				s.Require().NotNil(args, "Counter create arguments should not be nil")

				var counterValue int64
				foundValue := false
				for _, field := range args.GetFields() {
					if field.GetLabel() == "value" {
						counterValue = field.GetValue().GetInt64()
						foundValue = true
						break
					}
				}
				s.Require().True(foundValue, "Counter 'value' field not found in create arguments")
				s.Require().Equal(expectedValue, counterValue, "Counter value should be %d", expectedValue)
				s.T().Logf("Counter value verified: %d", counterValue)
				return
			}
		}
	}
	s.Fail("Counter contract not found in transaction events")
}
