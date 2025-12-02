//go:build e2e

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	aptose2e "github.com/smartcontractkit/mcms/e2e/tests/aptos"
	evme2e "github.com/smartcontractkit/mcms/e2e/tests/evm"
	solanae2e "github.com/smartcontractkit/mcms/e2e/tests/solana"
	suie2e "github.com/smartcontractkit/mcms/e2e/tests/sui"
	tone2e "github.com/smartcontractkit/mcms/e2e/tests/ton"
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

func TestAptosSuite(t *testing.T) {
	suite.Run(t, new(aptose2e.AptosTestSuite))
}

func TestSuiSuite(t *testing.T) {
	suite.Run(t, new(suie2e.TimelockProposalTestSuite))
	suite.Run(t, new(suie2e.InspectionTestSuite))
	suite.Run(t, new(suie2e.TimelockInspectionTestSuite))
	suite.Run(t, new(suie2e.SetRootTestSuite))
	suite.Run(t, new(suie2e.MCMSUserTestSuite))
	suite.Run(t, new(suie2e.TimelockCancelProposalTestSuite))
	suite.Run(t, new(suie2e.MCMSUserUpgradeTestSuite))
}

func TestTONSuite(t *testing.T) {
	suite.Run(t, new(tone2e.SigningTestSuite))
	suite.Run(t, new(tone2e.SetConfigTestSuite))
	suite.Run(t, new(tone2e.SetRootTestSuite))
	suite.Run(t, new(tone2e.InspectionTestSuite))
	suite.Run(t, new(tone2e.TimelockInspectionTestSuite))
}
