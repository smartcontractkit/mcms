//go:build e2e

package sui

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
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
	mcmsUserPackageId     string
	mcmsUser              modulemcmsuser.IMcmsUser
	mcmsUserOwnerCapObj   string
	mcmsUserUpgradeCapObj string

	// State Object passed into `mcms_entrypoint`
	stateObj string
}

func (s *SuiTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	var testSigner bindutils.SuiSigner
	if s.SuiBlockchain != nil {
		// Use mnemonic from blockchain network (existing behavior)
		account := s.SuiBlockchain.NetworkSpecificData.SuiAccount

		// Create a Sui signer from the mnemonic using the block-vision SDK
		signerAccount, err := signer.NewSignertWithMnemonic(account.Mnemonic)
		s.Require().NoError(err, "Failed to create signer from mnemonic")

		// Get the private key from the signer
		privateKey := signerAccount.PriKey
		testSigner = NewTestPrivateKeySigner(privateKey)
	} else {
		// Use private key from config (local node scenario)
		s.Require().NotEmpty(s.TestSetup.Config.Settings.PrivateKeys, "No private keys available in config")

		privateKeyHex := s.TestSetup.Config.Settings.PrivateKeys[0]
		if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
			privateKeyHex = privateKeyHex[2:]
		}

		privateKeyBytes, err := hex.DecodeString(privateKeyHex)
		s.Require().NoError(err, "Failed to decode private key hex")
		s.Require().Equal(32, len(privateKeyBytes), "Private key seed must be 32 bytes")

		// ed25519.NewKeyFromSeed creates a proper 64-byte private key from 32-byte seed
		privateKey := ed25519.NewKeyFromSeed(privateKeyBytes)
		testSigner = NewTestPrivateKeySigner(privateKey)

		// Fund the account from local faucet when using local node
		if s.TestSetup.Config.Settings.LocalSuiNodeURL != "" {
			address, err := testSigner.GetAddress()
			s.Require().NoError(err, "Failed to get address from signer")

			s.T().Logf("Funding account %s from local faucet", address)
			err = s.fundAccountFromLocalFaucet(address)
			if err != nil {
				s.T().Logf("Warning: Failed to fund account from faucet: %v", err)
				s.T().Logf("You may need to fund the account manually or ensure the local faucet is running")
			} else {
				// Wait a moment for the funding transaction to be processed
				time.Sleep(2 * time.Second)
			}
		}
	}

	// Set up Sui client
	s.client = s.SuiClient
	s.signer = testSigner
	s.chainSelector = types.ChainSelector(cselectors.SUI_TESTNET.Selector)
}

// FaucetRequest represents the request body for the local Sui faucet
type FaucetRequest struct {
	FixedAmountRequest struct {
		Recipient string `json:"recipient"`
	} `json:"FixedAmountRequest"`
}

// Call if running against a local Sui node to fund the test account
func (s *SuiTestSuite) fundAccountFromLocalFaucet(address string) error {
	faucetURLs := []string{
		"http://127.0.0.1:9123/gas", // Default sui local network faucet
		"http://127.0.0.1:5003/gas", // Alternative faucet endpoint
	}

	request := FaucetRequest{}
	request.FixedAmountRequest.Recipient = address

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal faucet request: %w", err)
	}

	var lastErr error
	for _, faucetURL := range faucetURLs {
		s.T().Logf("Attempting to fund address %s from faucet %s", address, faucetURL)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(faucetURL, "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			lastErr = fmt.Errorf("failed to request from faucet %s: %w", faucetURL, err)
			s.T().Logf("Faucet request failed: %v", lastErr)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			s.T().Logf("Successfully funded address %s from faucet %s", address, faucetURL)
			return nil
		}

		lastErr = fmt.Errorf("faucet %s returned status %d", faucetURL, resp.StatusCode)
		s.T().Logf("Faucet request failed: %v", lastErr)
	}

	return fmt.Errorf("all faucet attempts failed, last error: %w", lastErr)
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

	s.mcmsUserPackageId = mcmsUserPackage.Address()
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

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
