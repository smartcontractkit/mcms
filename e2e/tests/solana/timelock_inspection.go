//go:build e2e
// +build e2e

package solanae2e

import (
	"context"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

var testPDASeedTimelockInspection = [32]byte{'t', 'e', 's', 't', '-', 't', 'i', 'm', 'e', 'i', 'n', 's', 'p', 'e', 'c', 't'}

func (s *SolanaTestSuite) TestGetProposers() {
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	proposers, err := inspector.GetProposers(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockInspection))
	s.Require().NoError(err, "Failed to get proposers")
	s.Require().Len(proposers, 2, "Expected 2 proposers")

	var expected = make([]string, 0, len(s.Roles[timelock.Proposer_Role].Accounts))
	for _, acc := range s.Roles[timelock.Proposer_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, proposers, "Proposers don't match")
}

func (s *SolanaTestSuite) TestGetExecutors() {
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	executors, err := inspector.GetExecutors(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockInspection))
	s.Require().NoError(err, "Failed to get executors")
	s.Require().Len(executors, 2, "Expected 2 executors")

	var expected = make([]string, 0, len(s.Roles[timelock.Executor_Role].Accounts))
	for _, acc := range s.Roles[timelock.Executor_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, executors, "Executors don't match")
}

func (s *SolanaTestSuite) TestGetCancellers() {
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	cancellers, err := inspector.GetCancellers(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockInspection))
	s.Require().NoError(err, "Failed to get cancellers")
	s.Require().Len(cancellers, 2, "Expected 2 cancellers")

	var expected = make([]string, 0, len(s.Roles[timelock.Canceller_Role].Accounts))
	for _, acc := range s.Roles[timelock.Canceller_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, cancellers, "Cancellers don't match")
}

func (s *SolanaTestSuite) TestGetBypassers() {
	ctx := context.Background()

	inspector := solanasdk.NewTimelockInspector(s.SolanaClient)
	bypassers, err := inspector.GetBypassers(ctx, solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockInspection))
	s.Require().NoError(err, "Failed to get bypassers")
	s.Require().Len(bypassers, 2, "Expected 2 bypassers")

	var expected = make([]string, 0, len(s.Roles[timelock.Bypasser_Role].Accounts))
	for _, acc := range s.Roles[timelock.Bypasser_Role].Accounts {
		expected = append(expected, acc.PublicKey().String())
	}
	s.Require().ElementsMatch(expected, bypassers, "Bypassers don't match")
}
