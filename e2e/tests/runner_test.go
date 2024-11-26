//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// Run the test suite
func TestE2ESuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(TimelockInspectionTestSuite))
	suite.Run(t, new(InspectionTestSuite))
	suite.Run(t, new(ExecutionTestSuite))
}
