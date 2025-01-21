//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"time"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

var testPDASeedTimelockGetProposers = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'p', 'r', 'o', 'p', 'o', 's', 'e', 'r'}
var testPDASeedTimelockGetExecutors = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'e', 'x', 'e', 'c'}
var testPDASeedTimelockGetCancellers = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'c', 'a', 'n', 'c', 'e', 'l'}
var testPDASeedTimelockGetBypassers = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'b', 'y', 'p', 'a', 's', 's'}

func (s *SolanaTestSuite) TestGetProposers() {
	s.SetupTimelockWorker(testPDASeedTimelockGetProposers, 1*time.Second)
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

func (s *SolanaTestSuite) TestGetExecutors() {
	s.SetupTimelockWorker(testPDASeedTimelockGetExecutors, 1*time.Second)
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

func (s *SolanaTestSuite) TestGetCancellers() {
	s.SetupTimelockWorker(testPDASeedTimelockGetCancellers, 1*time.Second)
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

func (s *SolanaTestSuite) TestGetBypassers() {
	s.SetupTimelockWorker(testPDASeedTimelockGetBypassers, 1*time.Second)
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
