//go:build e2e

package sui

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuiE2E(t *testing.T) {
	suite.Run(t, new(SuiTestSuite))
}
