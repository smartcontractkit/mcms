//go:build e2e

package sui

import (
	"context"

	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	"github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
)

type SuiTestSuite struct {
	suite.Suite
	e2e.TestSetup

	client sui.ISuiAPI
	signer bindutils.SuiSigner

	mcms        module_mcms.IMcms
	mcmsObject  string
	timelockObj string
	depStateObj string
	registryObj string
	accountObj  string
	ownerCapObj string
}

func (s *SuiTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	// Set up Sui client
	s.client = s.TestSetup.SuiClient
	// TODO: Find funded accounts
	// s.signer = signer
	s.deployMCMSContract()
}

func (s *SuiTestSuite) deployMCMSContract() {

	mcmsPackage, tx, err := mcms.PublishMCMS(context.Background(), &bind.CallOpts{
		Signer: s.signer,
	}, s.client)
	s.Require().NoError(err, "Failed to publish MCMS package")
	s.mcms = mcmsPackage.MCMS()

	mcmsObject, err1 := bind.FindObjectIdFromPublishTx(*tx, "mcms", "MultisigState")
	timelockObj, err2 := bind.FindObjectIdFromPublishTx(*tx, "mcms", "Timelock")
	depState, err3 := bind.FindObjectIdFromPublishTx(*tx, "mcms_deployer", "DeployerState")
	reg, err4 := bind.FindObjectIdFromPublishTx(*tx, "mcms_registry", "Registry")
	acc, err5 := bind.FindObjectIdFromPublishTx(*tx, "mcms_account", "AccountState")
	ownCap, err6 := bind.FindObjectIdFromPublishTx(*tx, "mcms_account", "OwnerCap")

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
		s.T().Fatalf("Failed to find object IDs in publish tx: %v, %v, %v, %v, %v, %v", err1, err2, err3, err4, err5, err6)
	}

	s.mcmsObject = mcmsObject
	s.timelockObj = timelockObj
	s.depStateObj = depState
	s.registryObj = reg
	s.accountObj = acc
	s.ownerCapObj = ownCap
}
