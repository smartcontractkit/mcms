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

type MCMSTimelockInspectionTestSuite struct {
	TestSuite

	// Test signers
	signers       []*ecdsa.PrivateKey
	signerAddrs   []common.Address
	sortedSigners []*ecdsa.PrivateKey

	// Counter contract for testing
	counterInstanceID string
	counterCID        string
}

func TestMCMSTimelockInspectionTestSuite(t *testing.T) {
	suite.Run(t, new(MCMSTimelockInspectionTestSuite))
}

// SetupSuite runs before the test suite
func (s *MCMSTimelockInspectionTestSuite) SetupSuite() {
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
	s.signerAddrs = make([]common.Address, len(s.sortedSigners))
	for i, signer := range s.sortedSigners {
		s.signerAddrs[i] = crypto.PubkeyToAddress(signer.PublicKey)
	}

	// Deploy MCMS with config
	config := s.create2of3Config()
	s.DeployMCMSWithConfig(config)
}

func (s *MCMSTimelockInspectionTestSuite) create2of3Config() *mcmstypes.Config {
	return &mcmstypes.Config{
		Quorum:  2,
		Signers: s.signerAddrs,
	}
}

// deployCounterContract deploys a Counter contract for testing
func (s *MCMSTimelockInspectionTestSuite) deployCounterContract() {
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

func (s *MCMSTimelockInspectionTestSuite) TestGetProposers() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	proposers, err := inspector.GetProposers(ctx, s.mcmsContractID)
	s.Require().NoError(err, "GetProposers should not return an error")
	s.Require().Equal(len(s.signerAddrs), len(proposers), "Should have correct number of proposers")
}

func (s *MCMSTimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Canton doesn't have a separate executor role
	executors, err := inspector.GetExecutors(ctx, s.mcmsContractID)
	s.Require().Error(err, "GetExecutors should return an error on Canton")
	s.Require().Contains(err.Error(), "unsupported")
	s.Require().Nil(executors, "Executors should be nil when unsupported")
}

func (s *MCMSTimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Bypassers should return empty if not configured
	bypassers, err := inspector.GetBypassers(ctx, s.mcmsContractID)
	s.Require().NoError(err, "GetBypassers should not return an error")
	// Initially no bypassers configured
	s.T().Logf("GetBypassers returned %d bypassers", len(bypassers))
}

func (s *MCMSTimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Cancellers should return empty if not configured
	cancellers, err := inspector.GetCancellers(ctx, s.mcmsContractID)
	s.Require().NoError(err, "GetCancellers should not return an error")
	// Initially no cancellers configured
	s.T().Logf("GetCancellers returned %d cancellers", len(cancellers))
}

func (s *MCMSTimelockInspectionTestSuite) TestIsOperationWithNonExistentId() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-non-existent-operation-id")

	isOp, err := inspector.IsOperation(ctx, s.mcmsContractID, randomOpID)
	s.Require().NoError(err, "IsOperation should not return an error")
	s.Require().False(isOp, "Random operation ID should not exist")
}

func (s *MCMSTimelockInspectionTestSuite) TestIsOperationPendingWithNonExistentId() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-pending-operation-id")

	isPending, err := inspector.IsOperationPending(ctx, s.mcmsContractID, randomOpID)
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().False(isPending, "Random operation ID should not be pending")
}

func (s *MCMSTimelockInspectionTestSuite) TestIsOperationReadyWithNonExistentId() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-ready-operation-id")

	isReady, err := inspector.IsOperationReady(ctx, s.mcmsContractID, randomOpID)
	s.Require().NoError(err, "IsOperationReady should not return an error")
	s.Require().False(isReady, "Random operation ID should not be ready")
}

func (s *MCMSTimelockInspectionTestSuite) TestIsOperationDoneWithNonExistentId() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-done-operation-id")

	isDone, err := inspector.IsOperationDone(ctx, s.mcmsContractID, randomOpID)
	s.Require().NoError(err, "IsOperationDone should not return an error")
	s.Require().False(isDone, "Random operation ID should not be done")
}

func (s *MCMSTimelockInspectionTestSuite) TestGetMinDelay() {
	ctx := s.T().Context()

	inspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	minDelay, err := inspector.GetMinDelay(ctx, s.mcmsContractID)
	s.Require().NoError(err, "GetMinDelay should not return an error")
	// The default minDelay is 0 (we set it to 0 microseconds in the test setup)
	s.T().Logf("GetMinDelay returned: %d seconds", minDelay)
}

func (s *MCMSTimelockInspectionTestSuite) TestOperationLifecycle() {
	ctx := s.T().Context()

	// Deploy counter contract for this test
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
		SetDescription("Canton Timelock Operation Lifecycle test").
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

	// Sign with first 2 sorted signers
	_, _, err = s.SignProposal(&proposal, inspector, s.sortedSigners[:2], 2)
	s.Require().NoError(err)

	// Create executable
	executable, err := mcmscore.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	// Set the root
	txSetRoot, err := executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	// Update contract ID
	rawData, ok := txSetRoot.RawData.(map[string]any)
	s.Require().True(ok)
	newMCMSContractID, ok := rawData["NewMCMSContractID"].(string)
	s.Require().True(ok)
	s.mcmsContractID = newMCMSContractID

	// Update proposal metadata
	newMetadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, s.proposerMcmsId, s.mcmsId, s.mcmsContractID, false)
	s.Require().NoError(err)
	proposal.ChainMetadata[s.chainSelector] = newMetadata

	// Execute (schedules the batch)
	txExecute, err := executable.Execute(ctx, 0)
	s.Require().NoError(err)

	// Update MCMS contract ID
	rawTxData, ok := txExecute.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawTxData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	// Create timelock inspector
	timelockInspector := cantonsdk.NewTimelockInspector(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)

	// Create timelock executor and executable
	timelockExecutor := cantonsdk.NewTimelockExecutor(s.participant.StateServiceClient, s.participant.CommandServiceClient, s.participant.UserName, s.participant.Party)
	timelockExecutors := map[mcmstypes.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	}
	timelockExecutable, err := mcmscore.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	// Get operation ID
	scheduledOpID, err := timelockExecutable.GetOpID(ctx, 0, batchOp, s.chainSelector)
	s.Require().NoError(err)

	// STATE 1: After scheduling - operation should exist and be pending
	isOperation, err := timelockInspector.IsOperation(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isOperation, "Operation should exist after scheduling")

	isPending, err := timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isPending, "Operation should be pending after scheduling")

	isDone, err := timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isDone, "Operation should not be done after scheduling")

	s.T().Log("STATE 1 verified: Operation exists, is pending, not done")

	// Wait for delay
	time.Sleep(2 * time.Second)

	// STATE 2: After delay - operation should be ready
	isReady, err := timelockInspector.IsOperationReady(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isReady, "Operation should be ready after delay")

	isPending, err = timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isPending, "Operation should still be pending (not executed yet)")

	s.T().Log("STATE 2 verified: Operation is ready and still pending")

	// Execute the scheduled batch
	txTimelockExec, err := timelockExecutable.Execute(ctx, 0, mcmscore.WithCallProxy(s.mcmsContractID))
	s.Require().NoError(err)

	// Update MCMS contract ID
	rawExecData, ok := txTimelockExec.RawData.(map[string]any)
	s.Require().True(ok)
	if newID, ok := rawExecData["NewMCMSContractID"].(string); ok && newID != "" {
		s.mcmsContractID = newID
	}

	// STATE 3: After execution - operation should be done
	isDone, err = timelockInspector.IsOperationDone(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isDone, "Operation should be done after execution")

	isPending, err = timelockInspector.IsOperationPending(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().False(isPending, "Operation should not be pending after execution")

	isOperation, err = timelockInspector.IsOperation(ctx, s.mcmsContractID, scheduledOpID)
	s.Require().NoError(err)
	s.Require().True(isOperation, "Operation should still exist after execution")

	s.T().Log("STATE 3 verified: Operation is done, not pending, still exists")

	s.T().Log("TestOperationLifecycle completed successfully")
}
