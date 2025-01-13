//go:build e2e
// +build e2e

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	e2e_solana "github.com/smartcontractkit/mcms/e2e/tests/solana"
)

//func TestEVMSuite(t *testing.T) {
//	suite.Run(t, new(e2e_evm.InspectionTestSuite))
//	suite.Run(t, new(e2e_evm.ExecutionTestSuite))
//	suite.Run(t, new(e2e_evm.TimelockInspectionTestSuite))
//	suite.Run(t, new(e2e_evm.SetRootTestSuite))
//	suite.Run(t, new(e2e_evm.SigningTestSuite))
//}

//go:generate ./solana/compile-mcm-contracts.sh
func TestSolanaSuite(t *testing.T) {
	suite.Run(t, new(e2e_solana.SolanaTestSuite))
}
