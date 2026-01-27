//go:build e2e

package canton

type MCMSConfigurerTestSuite struct {
	TestSuite
}

// SetupSuite runs before the test suite
func (s *MCMSConfigurerTestSuite) SetupSuite() {
	s.TestSuite.SetupSuite()
	s.DeployMCMSContract()

	s.TestSetConfig()
}

func (s *TestSuite) TestSetConfig() {
	// TODO
	s.Require().True(true)
}
