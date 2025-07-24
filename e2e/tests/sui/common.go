//go:build e2e

package sui

import (
	"context"

	"github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	module_mcms_account "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_account"

	"github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/types"
)

type SuiTestSuite struct {
	suite.Suite
	e2e.TestSetup

	client sui.ISuiAPI
	signer bindutils.SuiSigner

	chainSelector types.ChainSelector

	// MCMS
	mcmsPackageId string
	mcms          module_mcms.IMcms
	mcmsObj       string
	timelockObj   string
	depStateObj   string
	registryObj   string
	accountObj    string
	ownerCapObj   string

	// MCMS Account
	mcmsAccount module_mcms_account.IMcmsAccount
}

func (s *SuiTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	account := s.TestSetup.SuiBlockchain.NetworkSpecificData.SuiAccount

	// Create a Sui signer from the mnemonic using the block-vision SDK
	signerAccount, err := signer.NewSignertWithMnemonic(account.Mnemonic)
	s.Require().NoError(err, "Failed to create signer from mnemonic")

	// Get the private key from the signer
	privateKey := signerAccount.PriKey
	testSigner := NewTestPrivateKeySigner(privateKey)

	// Set up Sui client
	s.client = s.TestSetup.SuiClient
	// TODO: Find funded accounts
	s.signer = testSigner
	s.chainSelector = types.ChainSelector(18395503381733958356)
}

func (s *SuiTestSuite) DeployMCMSContract() {
	gasBudget := uint64(300_000_000)
	mcmsPackage, tx, err := mcms.PublishMCMS(context.Background(), &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}, s.client)
	s.Require().NoError(err, "Failed to publish MCMS package")
	s.mcmsPackageId = mcmsPackage.Address()
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

	s.mcmsObj = mcmsObject
	s.timelockObj = timelockObj
	s.depStateObj = depState
	s.registryObj = reg
	s.accountObj = acc
	s.ownerCapObj = ownCap

	s.mcmsAccount, err = module_mcms_account.NewMcmsAccount(s.mcmsPackageId, s.client)
	s.Require().NoError(err, "Failed to create MCMS account instance")

}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
