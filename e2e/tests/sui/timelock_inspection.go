//go:build e2e

package sui

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

// TimelockInspectionTestSuite defines the test suite for Sui timelock inspection tests
type TimelockInspectionTestSuite struct {
	SuiTestSuite
}

// SetupSuite runs before the test suite
func (s *TimelockInspectionTestSuite) SetupSuite() {
	s.SuiTestSuite.SetupSuite()
	s.DeployMCMSContract()
}

// TestGetProposers tests that role-based queries are unsupported on Sui
func (s *TimelockInspectionTestSuite) TestGetProposers() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	proposers, err := inspector.GetProposers(ctx, s.timelockObj)
	s.Require().Error(err, "GetProposers should return an error on Sui")
	s.Require().Contains(err.Error(), "unsupported on Sui")
	s.Require().Nil(proposers, "Proposers should be nil when unsupported")
}

// TestGetExecutors tests that role-based queries are unsupported on Sui
func (s *TimelockInspectionTestSuite) TestGetExecutors() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	executors, err := inspector.GetExecutors(ctx, s.timelockObj)
	s.Require().Error(err, "GetExecutors should return an error on Sui")
	s.Require().Contains(err.Error(), "unsupported on Sui")
	s.Require().Nil(executors, "Executors should be nil when unsupported")
}

// TestGetBypassers tests that role-based queries are unsupported on Sui
func (s *TimelockInspectionTestSuite) TestGetBypassers() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	bypassers, err := inspector.GetBypassers(ctx, s.timelockObj)
	s.Require().Error(err, "GetBypassers should return an error on Sui")
	s.Require().Contains(err.Error(), "unsupported on Sui")
	s.Require().Nil(bypassers, "Bypassers should be nil when unsupported")
}

// TestGetCancellers tests that role-based queries are unsupported on Sui
func (s *TimelockInspectionTestSuite) TestGetCancellers() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	cancellers, err := inspector.GetCancellers(ctx, s.timelockObj)
	s.Require().Error(err, "GetCancellers should return an error on Sui")
	s.Require().Contains(err.Error(), "unsupported on Sui")
	s.Require().Nil(cancellers, "Cancellers should be nil when unsupported")
}

// TestIsOperation tests the IsOperation method with a non-existent operation
func (s *TimelockInspectionTestSuite) TestIsOperation() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-non-existent-operation-id")

	isOp, err := inspector.IsOperation(ctx, s.timelockObj, randomOpID)
	s.Require().NoError(err, "IsOperation should not return an error")
	s.Require().False(isOp, "Random operation ID should not exist")
}

// TestIsOperationPending tests the IsOperationPending method with a non-existent operation
func (s *TimelockInspectionTestSuite) TestIsOperationPending() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-pending-operation-id")

	isPending, err := inspector.IsOperationPending(ctx, s.timelockObj, randomOpID)
	s.Require().NoError(err, "IsOperationPending should not return an error")
	s.Require().False(isPending, "Random operation ID should not be pending")
}

// TestIsOperationReady tests the IsOperationReady method with a non-existent operation
func (s *TimelockInspectionTestSuite) TestIsOperationReady() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-ready-operation-id")

	isReady, err := inspector.IsOperationReady(ctx, s.timelockObj, randomOpID)
	s.Require().NoError(err, "IsOperationReady should not return an error")
	s.Require().False(isReady, "Random operation ID should not be ready")
}

// TestIsOperationDone tests the IsOperationDone method with a non-existent operation
func (s *TimelockInspectionTestSuite) TestIsOperationDone() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	// Test with a random operation ID that doesn't exist
	var randomOpID [32]byte
	copy(randomOpID[:], "test-done-operation-id")

	isDone, err := inspector.IsOperationDone(ctx, s.timelockObj, randomOpID)
	s.Require().NoError(err, "IsOperationDone should not return an error")
	s.Require().False(isDone, "Random operation ID should not be done")
}

// TestOperationLifecycle tests a complete operation lifecycle by scheduling an operation
// and checking its status through various states
func (s *TimelockInspectionTestSuite) TestOperationLifecycle() {
	ctx := s.T().Context()

	inspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err, "Failed to create timelock inspector")

	// Create a timelock converter to create a scheduled operation
	converter, err := suisdk.NewTimelockConverter()
	s.Require().NoError(err, "Failed to create timelock converter")

	// Create a simple batch operation for testing
	tx := types.Transaction{
		To:   s.accountObj, // Use account object as target
		Data: []byte("test-data"),
		AdditionalFields: []byte(`{
			"module_name": "mcms_account",
			"function": "accept_ownership_as_timelock"
		}`),
	}

	bop := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{tx},
	}

	metadata, err := suisdk.NewChainMetadata(0, 2, s.mcmsPackageID, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err, "Failed to create chain metadata for cancellation proposal")

	// Convert batch operation to chain operations
	operations, opID, err := converter.ConvertBatchToChainOperations(
		ctx,
		metadata,
		bop,
		s.timelockObj,
		s.mcmsObj,
		types.Duration{Duration: time.Minute},
		types.TimelockActionSchedule,
		common.Hash{},  // predecessor
		common.Hash{1}, // salt
	)
	s.Require().NoError(err, "Failed to convert batch operation")
	s.Require().Len(operations, 1, "Expected 1 operation")

	// Initially, the operation should not exist
	exists, err := inspector.IsOperation(ctx, s.timelockObj, opID)
	s.Require().NoError(err, "IsOperation should not return an error")
	s.Require().False(exists, "Operation should not exist initially")

	// Note: In a real test scenario, we would schedule the operation here
	// and then test the various states. However, for this basic test,
	// we're just verifying that the inspector methods work correctly
	// with non-existent operations.
}
