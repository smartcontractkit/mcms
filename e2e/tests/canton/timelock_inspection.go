//go:build e2e

package canton

import (
	"github.com/ethereum/go-ethereum/common"

	cantonsdk "github.com/smartcontractkit/mcms/sdk/canton"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// TimelockInspectionTestSuite defines the test suite for Canton timelock inspection.
// With stub implementations, all methods return "unsupported on Canton".
type TimelockInspectionTestSuite struct {
	TestSuite
	inspector *cantonsdk.TimelockInspector
}

// SetupSuite runs before the test suite.
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()
	s.DeployMCMSContract()
	s.inspector = cantonsdk.NewTimelockInspector()
}

// TestGetProposers tests that GetProposers returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	ctx := s.T().Context()
	proposers, err := s.inspector.GetProposers(ctx, s.mcmsContractID)
	s.Require().Error(err, "GetProposers should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(proposers)
}

// TestGetExecutors tests that GetExecutors returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()
	executors, err := s.inspector.GetExecutors(ctx, s.mcmsContractID)
	s.Require().Error(err, "GetExecutors should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(executors)
}

// TestGetBypassers tests that GetBypassers returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()
	bypassers, err := s.inspector.GetBypassers(ctx, s.mcmsContractID)
	s.Require().Error(err, "GetBypassers should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(bypassers)
}

// TestGetCancellers tests that GetCancellers returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()
	cancellers, err := s.inspector.GetCancellers(ctx, s.mcmsContractID)
	s.Require().Error(err, "GetCancellers should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Nil(cancellers)
}

// TestIsOperation tests that IsOperation returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestIsOperation() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-operation-id")
	isOp, err := s.inspector.IsOperation(ctx, s.mcmsContractID, opID)
	s.Require().Error(err, "IsOperation should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().False(isOp)
}

// TestIsOperationPending tests that IsOperationPending returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestIsOperationPending() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-pending-id")
	isPending, err := s.inspector.IsOperationPending(ctx, s.mcmsContractID, opID)
	s.Require().Error(err, "IsOperationPending should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().False(isPending)
}

// TestIsOperationReady tests that IsOperationReady returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestIsOperationReady() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-ready-id")
	isReady, err := s.inspector.IsOperationReady(ctx, s.mcmsContractID, opID)
	s.Require().Error(err, "IsOperationReady should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().False(isReady)
}

// TestIsOperationDone tests that IsOperationDone returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestIsOperationDone() {
	ctx := s.T().Context()
	var opID [32]byte
	copy(opID[:], "test-done-id")
	isDone, err := s.inspector.IsOperationDone(ctx, s.mcmsContractID, opID)
	s.Require().Error(err, "IsOperationDone should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().False(isDone)
}

// TestGetMinDelay tests that GetMinDelay returns unsupported on Canton (stub).
func (s *TimelockInspectionTestSuite) TestGetMinDelay() {
	ctx := s.T().Context()
	delay, err := s.inspector.GetMinDelay(ctx, s.mcmsContractID)
	s.Require().Error(err, "GetMinDelay should return an error on Canton stub")
	s.Require().Contains(err.Error(), "unsupported on Canton")
	s.Require().Equal(uint64(0), delay)
}

// TestTimelockConverterNotImplemented tests that ConvertBatchToChainOperations returns not implemented (stub).
func (s *TimelockInspectionTestSuite) TestTimelockConverterNotImplemented() {
	ctx := s.T().Context()
	converter := cantonsdk.NewTimelockConverter()
	bop := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []mcmstypes.Transaction{},
	}
	metadata := mcmstypes.ChainMetadata{MCMAddress: s.mcmsContractID}
	ops, opID, err := converter.ConvertBatchToChainOperations(
		ctx,
		metadata,
		bop,
		s.mcmsContractID,
		s.mcmsContractID,
		mcmstypes.Duration{},
		mcmstypes.TimelockActionSchedule,
		common.Hash{},
		common.Hash{},
	)
	s.Require().Error(err, "ConvertBatchToChainOperations should return an error on Canton stub")
	s.Require().Contains(err.Error(), "not implemented on Canton")
	s.Require().Nil(ops)
	s.Require().Equal(common.Hash{}, opID)
}

// TestTimelockExecutorNotImplemented tests that Execute returns not implemented (stub).
func (s *TimelockInspectionTestSuite) TestTimelockExecutorNotImplemented() {
	ctx := s.T().Context()
	executor := cantonsdk.NewTimelockExecutor()
	bop := mcmstypes.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []mcmstypes.Transaction{},
	}
	res, err := executor.Execute(ctx, bop, s.mcmsContractID, common.Hash{}, common.Hash{})
	s.Require().Error(err, "Execute should return an error on Canton stub")
	s.Require().Contains(err.Error(), "not implemented on Canton")
	s.Require().Equal(mcmstypes.TransactionResult{}, res)
}
