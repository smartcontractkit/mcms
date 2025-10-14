//go:build e2e

package sui

import (
	"fmt"

	"github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/block-vision/sui-go-sdk/transaction"
	"github.com/stretchr/testify/suite"

	cselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	modulemcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	modulemcmsaccount "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_account"
	modulemcmsuser "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_user"
	"github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms"
	mcmsuser "github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms/mcms_user"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
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
	mcms          modulemcms.IMcms
	mcmsObj       string
	timelockObj   string
	depStateObj   string
	registryObj   string
	accountObj    string
	ownerCapObj   string

	// MCMS Account
	mcmsAccount modulemcmsaccount.IMcmsAccount

	// MCMS User
	mcmsUserPackageID     string
	mcmsUser              modulemcmsuser.IMcmsUser
	mcmsUserOwnerCapObj   string
	mcmsUserUpgradeCapObj string

	// State Object passed into `mcms_entrypoint`
	stateObj string

	// Entrypoint Arg Encoder
	entrypointArgEncoder suisdk.EntrypointArgEncoder
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
	mcmsPackage, tx, err := mcms.PublishMCMS(s.T().Context(), &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}, s.client)
	s.Require().NoError(err, "Failed to publish MCMS package")
	s.mcmsPackageID = mcmsPackage.Address()
	s.mcms = mcmsPackage.MCMS()

	mcmsObject, err := bind.FindObjectIdFromPublishTx(*tx, "mcms", "MultisigState")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	timelockObj, err := bind.FindObjectIdFromPublishTx(*tx, "mcms", "Timelock")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	depState, err := bind.FindObjectIdFromPublishTx(*tx, "mcms_deployer", "DeployerState")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	reg, err := bind.FindObjectIdFromPublishTx(*tx, "mcms_registry", "Registry")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	acc, err := bind.FindObjectIdFromPublishTx(*tx, "mcms_account", "AccountState")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	ownCap, err := bind.FindObjectIdFromPublishTx(*tx, "mcms_account", "OwnerCap")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")

	s.mcmsObj = mcmsObject
	s.timelockObj = timelockObj
	s.depStateObj = depState
	s.registryObj = reg
	s.accountObj = acc
	s.ownerCapObj = ownCap

	s.mcmsAccount, err = modulemcmsaccount.NewMcmsAccount(s.mcmsPackageID, s.client)
	s.Require().NoError(err, "Failed to create MCMS account instance")

	s.entrypointArgEncoder = &TestEntrypointArgEncoder{
		registryObj: s.registryObj,
		client:      s.client,
	}
}

func (s *SuiTestSuite) DeployMCMSUserContract() {
	gasBudget := uint64(300_000_000)
	signerAddress, err := s.signer.GetAddress()
	s.Require().NoError(err, "Failed to get address")

	mcmsUserPackage, tx, err := mcmsuser.PublishMCMSUser(s.T().Context(), &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}, s.client, s.mcmsPackageID, signerAddress)
	s.Require().NoError(err, "Failed to publish MCMS user package")

	s.mcmsUserPackageID = mcmsUserPackage.Address()
	s.mcmsUser = mcmsUserPackage.MCMSUser()

	userDataObj, err := bind.FindObjectIdFromPublishTx(*tx, "mcms_user", "UserData")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	mcmsUserOwnerCapObj, err := bind.FindObjectIdFromPublishTx(*tx, "ownable", "OwnerCap")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")
	mcmsUserUpgradeCapObj, err := bind.FindObjectIdFromPublishTx(*tx, "package", "UpgradeCap")
	s.Require().NoError(err, "Failed to find object IDs in publish tx")

	s.mcmsUserOwnerCapObj = mcmsUserOwnerCapObj
	s.stateObj = userDataObj
	s.mcmsUserUpgradeCapObj = mcmsUserUpgradeCapObj

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

func (s *SuiTestSuite) extractByteArgsFromEncodedCall(encodedCall bind.EncodedCall) []byte {
	var args []byte
	for _, callArg := range encodedCall.CallArgs {
		if callArg.CallArg.UnresolvedObject != nil {
			args = append(args, callArg.CallArg.UnresolvedObject.ObjectId[:]...)
		}
		if callArg.CallArg.Pure != nil {
			args = append(args, callArg.CallArg.Pure.Bytes...)
		}
	}

	return args
}

type TestEntrypointArgEncoder struct {
	registryObj string
	client      sui.ISuiAPI
}

func (e *TestEntrypointArgEncoder) EncodeEntryPointArg(executingCallbackParams *transaction.Argument, target, module, function, stateObjID string, data []byte) (*bind.EncodedCall, error) {
	// For simplicity, we only support mcms_user as target in this test encoder
	if module != "mcms_user" {
		return nil, fmt.Errorf("unsupported module: %s", module)
	}
	mcmsUser, err := mcmsuser.NewMCMSUser(target, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create entrypoint encoder (mcmsUser) contract: %w", err)
	}
	entryPointCall, err := mcmsUser.MCMSUser().Encoder().McmsFunctionOneWithArgs(
		stateObjID,
		e.registryObj,
		executingCallbackParams,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode entrypoint arg (mcmsUser): %w", err)
	}

	return entryPointCall, nil
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
