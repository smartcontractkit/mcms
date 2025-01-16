//go:build e2e
// +build e2e

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	evme2e "github.com/smartcontractkit/mcms/e2e/tests/evm"
	solanae2e "github.com/smartcontractkit/mcms/e2e/tests/solana"
)

func TestEVMSuite(t *testing.T) {
	suite.Run(t, new(evme2e.InspectionTestSuite))
	suite.Run(t, new(evme2e.ExecutionTestSuite))
	suite.Run(t, new(evme2e.TimelockInspectionTestSuite))
	suite.Run(t, new(evme2e.SetRootTestSuite))
	suite.Run(t, new(evme2e.SigningTestSuite))
}

//go:generate ./solana/compile-mcm-contracts.sh
func TestSolanaSuite(t *testing.T) {
	// Initialize SolanaTestSuite as a pointer
	baseSuite := &solanae2e.SolanaTestSuite{}

	// Run tests that depend on SolanaTestSuite
	suite.Run(t, baseSuite)

	// Use the pointer directly when initializing dependent test suites
	suite.Run(t, &solanae2e.ExecuteSolanaTestSuite{SolanaTestSuite: baseSuite})
	suite.Run(t, &solanae2e.InspectSolanaTestSuite{SolanaTestSuite: baseSuite})
	suite.Run(t, &solanae2e.SetConfigSolanaTestSuite{SolanaTestSuite: baseSuite})
}
