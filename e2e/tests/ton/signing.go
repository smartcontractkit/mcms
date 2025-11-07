//go:build e2e
// +build e2e

package tone2e

import (
	"github.com/smartcontractkit/mcms/e2e/tests/common"
)

// SigningTestSuite tests signing a proposal and converting back to a file
type SigningTestSuite struct {
	common.SigningTestSuite
}
