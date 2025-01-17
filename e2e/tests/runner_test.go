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
	suite.Run(t, new(solanae2e.SolanaTestSuite))
}
