//go:build e2e

package solanae2e

import (
	"context"
	"time"

	"github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
	timelockutils "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/timelock"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

var testPDASeedTimelockGetProposers = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'p', 'r', 'o', 'p', 'o', 's', 'e', 'r'}
var testPDASeedTimelockGetExecutors = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'e', 'x', 'e', 'c'}
var testPDASeedTimelockGetCancellers = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'c', 'a', 'n', 'c', 'e', 'l'}
var testPDASeedTimelockGetBypassers = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'b', 'y', 'p', 'a', 's', 's'}
var testPDASeedTimelockIsOperations = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'o', 'p', 's'}
var testPDASeedTimelockIsOperationsPending = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'o', 'p', 's', 'p', 'e', 'n', 'd'}
var testPDASeedTimelockIsOperationsReady = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'o', 'p', 's', 'r', 'e', 'a', 'd', 'y'}
var testPDASeedTimelockIsOperationsDone = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'o', 'p', 's', 'd', 'o', 'n', 'e'}
var testPDASeedTimelockMinDelay = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'm', 'i', 'n', 'd', 'e', 'l', 'a', 'y'}

func (s *TestSuite) TestGetProposers() {
	s.SetupTimelock(testPDASeedTimelockGetProposers, 1*time.Second)
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	proposers, err := inspector.GetProposers(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockGetProposers))
	s.Require().NoError(err, "Failed to get proposers")
	s.Require().Len(proposers, 2, "Expected 2 proposers")

	var expected = make([]string, 0, len(s.Roles[timelock.Proposer_Role].Accounts))
	for _, acc := range s.Roles[timelock.Proposer_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, proposers, "Proposers don't match")
}

func (s *TestSuite) TestGetExecutors() {
	s.SetupTimelock(testPDASeedTimelockGetExecutors, 1*time.Second)
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	executors, err := inspector.GetExecutors(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockGetExecutors))
	s.Require().NoError(err, "Failed to get executors")
	s.Require().Len(executors, 2, "Expected 2 executors")

	var expected = make([]string, 0, len(s.Roles[timelock.Executor_Role].Accounts))
	for _, acc := range s.Roles[timelock.Executor_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, executors, "Executors don't match")
}

func (s *TestSuite) TestGetCancellers() {
	s.SetupTimelock(testPDASeedTimelockGetCancellers, 1*time.Second)
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	cancellers, err := inspector.GetCancellers(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockGetCancellers))
	s.Require().NoError(err, "Failed to get cancellers")
	s.Require().Len(cancellers, 2, "Expected 2 cancellers")

	var expected = make([]string, 0, len(s.Roles[timelock.Canceller_Role].Accounts))
	for _, acc := range s.Roles[timelock.Canceller_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, cancellers, "Cancellers don't match")
}

func (s *TestSuite) TestGetBypassers() {
	s.SetupTimelock(testPDASeedTimelockGetBypassers, 1*time.Second)
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	bypassers, err := inspector.GetBypassers(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockGetBypassers))
	s.Require().NoError(err, "Failed to get bypassers")
	s.Require().Len(bypassers, 2, "Expected 2 bypassers")

	var expected = make([]string, 0, len(s.Roles[timelock.Bypasser_Role].Accounts))
	for _, acc := range s.Roles[timelock.Bypasser_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, bypassers, "Bypassers don't match")
}

func (s *TestSuite) TestIsOperation() {
	ctx := context.Background()
	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	s.SetupTimelock(testPDASeedTimelockIsOperations, 1*time.Second)
	op := s.createOperation(testPDASeedTimelockIsOperations)
	s.initOperation(ctx, op, testPDASeedTimelockIsOperations, admin)

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	operation, err := inspector.IsOperation(ctx,
		solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockIsOperations), op.OperationID())

	s.Require().NoError(err, "Failed to check if operation exists")
	s.Require().True(operation, "Operation should exist")
}

func (s *TestSuite) TestIOperationPending() {
	ctx := context.Background()
	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	s.SetupTimelock(testPDASeedTimelockIsOperationsPending, 1*time.Second)
	op := s.createOperation(testPDASeedTimelockIsOperationsPending)
	s.initOperation(ctx, op, testPDASeedTimelockIsOperationsPending, admin)
	s.scheduleOperation(ctx, testPDASeedTimelockIsOperationsPending, 1*time.Second, op.OperationID())

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	operation, err := inspector.IsOperationPending(ctx,
		solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockIsOperationsPending), op.OperationID())

	s.Require().NoError(err, "Failed to check if operation is pending")
	s.Require().True(operation, "Operation should be pending")
}

func (s *TestSuite) TestIsOperationReady() {
	ctx := context.Background()
	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	s.SetupTimelock(testPDASeedTimelockIsOperationsReady, 1*time.Second)
	op := s.createOperation(testPDASeedTimelockIsOperationsReady)
	s.initOperation(ctx, op, testPDASeedTimelockIsOperationsReady, admin)
	s.scheduleOperation(ctx, testPDASeedTimelockIsOperationsReady, 1*time.Second, op.OperationID())

	s.waitForOperationToBeReady(ctx, testPDASeedTimelockIsOperationsReady, op.OperationID())

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	operation, err := inspector.IsOperationReady(ctx,
		solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockIsOperationsReady), op.OperationID())

	s.Require().NoError(err, "Failed to check if operation is ready")
	s.Require().True(operation, "Operation should be ready")
}

func (s *TestSuite) TestIsOperationDone() {
	ctx := context.Background()
	admin, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	s.SetupTimelock(testPDASeedTimelockIsOperationsDone, 1*time.Second)
	op := s.createOperation(testPDASeedTimelockIsOperationsDone)
	s.initOperation(ctx, op, testPDASeedTimelockIsOperationsDone, admin)
	s.scheduleOperation(ctx, testPDASeedTimelockIsOperationsDone, 1*time.Second, op.OperationID())
	s.waitForOperationToBeReady(ctx, testPDASeedTimelockIsOperationsDone, op.OperationID())
	s.executeOperation(ctx, testPDASeedTimelockIsOperationsDone, op.OperationID())

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	operation, err := inspector.IsOperationDone(ctx,
		solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockIsOperationsDone), op.OperationID())

	s.Require().NoError(err, "Failed to check if operation is done")
	s.Require().True(operation, "Operation should be done")
}

func (s *TestSuite) TestGetMinDelay() {
	ctx := context.Background()
	minDelay := 1 * time.Second
	s.SetupTimelock(testPDASeedTimelockMinDelay, minDelay)

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	delay, err := inspector.GetMinDelay(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockMinDelay))

	s.Require().NoError(err, "Failed to query min delay")
	s.Require().Equal(delay, uint64(minDelay.Seconds()), "Min delay should match the configured value")
}

func (s *TestSuite) createOperation(timelockID [32]byte) timelockutils.Operation {
	salt, serr := timelockutils.SimpleSalt()
	s.Require().NoError(serr)

	op := timelockutils.Operation{
		TimelockID:  timelockID,
		Predecessor: [32]byte{},
		Salt:        salt,
	}

	return op
}
