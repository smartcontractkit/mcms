//go:build e2e

package sui

import (
	"context"

	"github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/stretchr/testify/suite"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	moduleMcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	moduleMcmsAccount "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_account"
	moduleMcmsUser "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_user"
	"github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms"
	mcmsuser "github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms/mcms_user"
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
	mcmsPackageID string
	mcms          moduleMcms.IMcms
	mcmsObj       string
	timelockObj   string
	depStateObj   string
	registryObj   string
	accountObj    string
	ownerCapObj   string

	// MCMS Account
	mcmsAccount moduleMcmsAccount.IMcmsAccount

	// MCMS User
	mcmsUserPackageId   string
	mcmsUser            moduleMcmsUser.IMcmsUser
	mcmsUserOwnerCapObj string

	// State Object passed into `mcms_entrypoint`
	stateObj string
}

func (s *SuiTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	account := s.SuiBlockchain.NetworkSpecificData.SuiAccount

	// Create a Sui signer from the mnemonic using the block-vision SDK
	signerAccount, err := signer.NewSignertWithMnemonic(account.Mnemonic)
	s.Require().NoError(err, "Failed to create signer from mnemonic")

	// Get the private key from the signer
	privateKey := signerAccount.PriKey
	testSigner := NewTestPrivateKeySigner(privateKey)

	// Set up Sui client
	s.client = s.SuiClient
	// TODO: Find funded accounts
	s.signer = testSigner
	s.chainSelector = types.ChainSelector(cselectors.SUI_TESTNET.Selector)
}

func (s *SuiTestSuite) DeployMCMSContract() {
	gasBudget := uint64(300_000_000)
	mcmsPackage, tx, err := mcms.PublishMCMS(context.Background(), &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}, s.client)
	s.Require().NoError(err, "Failed to publish MCMS package")
	s.mcmsPackageID = mcmsPackage.Address()
	s.mcms = mcmsPackage.MCMS()

	mcmsObject, err1 := bind.FindObjectIdFromPublishTx(*tx, "mcms", "MultisigState")
	timelockObj, err2 := bind.FindObjectIdFromPublishTx(*tx, "mcms", "Timelock")
	depState, err3 := bind.FindObjectIdFromPublishTx(*tx, "mcms_deployer", "DeployerState")
	reg, err4 := bind.FindObjectIdFromPublishTx(*tx, "mcms_registry", "Registry")
	acc, err5 := bind.FindObjectIdFromPublishTx(*tx, "mcms_account", "AccountState")
	ownCap, err6 := bind.FindObjectIdFromPublishTx(*tx, "mcms_account", "OwnerCap")

	s.Require().NoError(err1, "Failed to find object IDs in publish tx")
	s.Require().NoError(err2, "Failed to find object IDs in publish tx")
	s.Require().NoError(err3, "Failed to find object IDs in publish tx")
	s.Require().NoError(err4, "Failed to find object IDs in publish tx")
	s.Require().NoError(err5, "Failed to find object IDs in publish tx")
	s.Require().NoError(err6, "Failed to find object IDs in publish tx")

	s.mcmsObj = mcmsObject
	s.timelockObj = timelockObj
	s.depStateObj = depState
	s.registryObj = reg
	s.accountObj = acc
	s.ownerCapObj = ownCap

	s.mcmsAccount, err = moduleMcmsAccount.NewMcmsAccount(s.mcmsPackageID, s.client)
	s.Require().NoError(err, "Failed to create MCMS account instance")
}

func (s *SuiTestSuite) DeployMCMSUserContract() {
	gasBudget := uint64(300_000_000)
	signerAddress, err := s.signer.GetAddress()
	s.Require().NoError(err, "Failed to get address")

	mcmsUserPackage, tx, err := mcmsuser.PublishMCMSUser(context.Background(), &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}, s.client, s.mcmsPackageID, signerAddress)
	s.Require().NoError(err, "Failed to publish MCMS user package")

	s.mcmsUserPackageId = mcmsUserPackage.Address()
	s.mcmsUser = mcmsUserPackage.MCMSUser()

	userDataObj, err1 := bind.FindObjectIdFromPublishTx(*tx, "mcms_user", "UserData")
	mcmsUserOwnerCapObj, err2 := bind.FindObjectIdFromPublishTx(*tx, "mcms_user", "OwnerCap")

	s.Require().NoError(err1, "Failed to find object IDs in publish tx")
	s.Require().NoError(err2, "Failed to find object IDs in publish tx")

	s.mcmsUserOwnerCapObj = mcmsUserOwnerCapObj
	s.stateObj = userDataObj

	// For executing, We need to register OwnerCap with MCMS
	{
		tx, err := s.mcmsUser.RegisterMcmsEntrypoint(
			s.T().Context(),
			&bind.CallOpts{
				Signer:           s.signer,
				WaitForExecution: true,
			},
			bind.Object{Id: s.mcmsUserOwnerCapObj},
			bind.Object{Id: s.registryObj},
			bind.Object{Id: s.stateObj},
		)
		s.Require().NoError(err, "Failed to register with MCMS")
		s.Require().NotEmpty(tx, "Transaction should not be empty")

		s.T().Logf("✅ Registered with MCMS in tx: %s", tx.Digest)
	}
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
