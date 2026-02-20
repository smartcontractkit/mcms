//go:build e2e

package canton

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"

	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// TimelockInspectionTestSuite defines the test suite for Canton timelock inspection.
type TimelockInspectionTestSuite struct {
	TestSuite
	inspector *cantonsdk.TimelockInspector
}

// SetupSuite runs before the test suite.
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()
	s.DeployMCMSContract()
	mcmsPkgID := ""
	if len(s.packageIDs) > 0 {
		mcmsPkgID = s.packageIDs[0]
	}
	s.inspector = cantonsdk.NewTimelockInspector(s.participant.CommandServiceClient, s.participant.StateServiceClient, s.participant.Party, mcmsPkgID)
}

// TestGetProposers tests that GetProposers returns unsupported on Canton.
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	ctx := s.T().Context()
	proposers, err := s.inspector.GetProposers(ctx, s.mcmsInstanceAddress)
	s.Require().Error(err, "GetProposers should return an error on Canton")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(proposers)
}

// TestGetExecutors tests that GetExecutors returns unsupported on Canton.
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()
	executors, err := s.inspector.GetExecutors(ctx, s.mcmsInstanceAddress)
	s.Require().Error(err, "GetExecutors should return an error on Canton")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(executors)
}

// TestGetBypassers tests that GetBypassers returns unsupported on Canton.
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()
	bypassers, err := s.inspector.GetBypassers(ctx, s.mcmsInstanceAddress)
	s.Require().Error(err, "GetBypassers should return an error on Canton")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(bypassers)
}

// TestGetCancellers tests that GetCancellers returns unsupported on Canton.
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()
	cancellers, err := s.inspector.GetCancellers(ctx, s.mcmsInstanceAddress)
	s.Require().Error(err, "GetCancellers should return an error on Canton")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(cancellers)
}

// TestIsOperation tests that IsOperation queries the ledger (returns false for unknown op ID).
func (s *TimelockInspectionTestSuite) TestIsOperation() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-operation-id")
	isOp, err := s.inspector.IsOperation(ctx, s.mcmsInstanceAddress, opID)
	s.Require().NoError(err)
	s.Require().False(isOp)
}

// TestIsOperationPending tests that IsOperationPending queries the ledger.
func (s *TimelockInspectionTestSuite) TestIsOperationPending() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-pending-id")
	isPending, err := s.inspector.IsOperationPending(ctx, s.mcmsInstanceAddress, opID)
	s.Require().NoError(err)
	s.Require().False(isPending)
}

// TestIsOperationReady tests that IsOperationReady queries the ledger.
func (s *TimelockInspectionTestSuite) TestIsOperationReady() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-ready-id")
	isReady, err := s.inspector.IsOperationReady(ctx, s.mcmsInstanceAddress, opID)
	s.Require().NoError(err)
	s.Require().False(isReady)
}

// TestIsOperationDone tests that IsOperationDone queries the ledger.
func (s *TimelockInspectionTestSuite) TestIsOperationDone() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-done-id")
	isDone, err := s.inspector.IsOperationDone(ctx, s.mcmsInstanceAddress, opID)
	s.Require().NoError(err)
	s.Require().False(isDone)
}

// TestGetMinDelay tests that GetMinDelay returns the MCMS min delay from the ledger.
func (s *TimelockInspectionTestSuite) TestGetMinDelay() {
	ctx := s.T().Context()
	delay, err := s.inspector.GetMinDelay(ctx, s.mcmsInstanceAddress)
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(delay, uint64(0))
}


// TestTimelockConverter tests that ConvertBatchToChainOperations returns one ScheduleBatch operation and a non-zero op ID.
func (s *TimelockInspectionTestSuite) TestTimelockConverter() {
	ctx := s.T().Context()
	metadata, err := cantonsdk.NewChainMetadata(0, 1, s.chainId, s.proposerMcmsId, s.mcmsInstanceAddress, false)
	s.Require().NoError(err)

	af := cantonsdk.AdditionalFields{
		TargetInstanceId: "instance@party",
		FunctionName:     "noop",
		OperationData:    "",
		TargetCid:        s.mcmsInstanceAddress,
		ContractIds:      []string{s.mcmsInstanceAddress},
	}
	afBytes, err := json.Marshal(af)
	s.Require().NoError(err)

	bop := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions: []mcmstypes.Transaction{{
			To:               s.mcmsInstanceAddress,
			Data:             []byte{},
			AdditionalFields: afBytes,
		}},
	}
	converter := cantonsdk.NewTimelockConverter()
	ops, opID, err := converter.ConvertBatchToChainOperations(
		ctx,
		metadata,
		bop,
		s.mcmsInstanceAddress,
		s.mcmsInstanceAddress,
		mcmstypes.NewDuration(0),
		mcmstypes.TimelockActionSchedule,
		common.Hash{},
		common.Hash{},
	)
	s.Require().NoError(err)
	s.Require().Len(ops, 1)
	s.Require().NotEqual(common.Hash{}, opID)
}

// TestTimelockExecutorExecuteEmptyBatch tests that Execute returns an error for empty batch.
func (s *TimelockInspectionTestSuite) TestTimelockExecutorExecuteEmptyBatch() {
	ctx := s.T().Context()
	mcmsPkgID := ""
	if len(s.packageIDs) > 0 {
		mcmsPkgID = s.packageIDs[0]
	}
	executor := cantonsdk.NewTimelockExecutor(s.participant.CommandServiceClient, s.participant.StateServiceClient, s.participant.Party, mcmsPkgID)
	bop := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []mcmstypes.Transaction{},
	}
	res, err := executor.Execute(ctx, bop, s.mcmsInstanceAddress, common.Hash{}, common.Hash{})
	s.Require().Error(err, "Execute should return an error for empty batch")
	s.Require().Contains(err.Error(), "no transactions")
	s.Require().Equal(mcmstypes.TransactionResult{}, res)
}
