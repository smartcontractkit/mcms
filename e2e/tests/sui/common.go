//go:build e2e

package sui

import (
	"context"
	"crypto/ed25519"

	"github.com/block-vision/sui-go-sdk/signer"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
	module_mcms "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms"
	module_mcms_account "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_account"
	module_mcms_user "github.com/smartcontractkit/chainlink-sui/bindings/generated/mcms/mcms_user"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms"
	mcms_user "github.com/smartcontractkit/chainlink-sui/bindings/packages/mcms/mcms_user"
	bindutils "github.com/smartcontractkit/chainlink-sui/bindings/utils"
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

	// MCMS User
	mcmsUserPackageId   string
	mcmsUser            module_mcms_user.IMcmsUser
	mcmsUserOwnerCapObj string

	// State Object passed into `mcms_entrypoint`
	stateObj string
}

type testSigner struct {
	privateKey ed25519.PrivateKey
}

/**
 * When running sui node locally, use this signer
 *
func newTestSigner() (*testSigner, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &testSigner{privateKey: privateKey}, nil
}

func (s *testSigner) Sign(message []byte) ([]string, error) {
	intentMessage := append([]byte{0x00, 0x00, 0x00}, message...)

	// Hash the message with blake2b
	hash := blake2b.Sum256(intentMessage)

	// Sign the hash
	signature := ed25519.Sign(s.privateKey, hash[:])

	// Get public key
	publicKey := s.privateKey.Public().(ed25519.PublicKey)

	// Create serialized signature: flag + signature + pubkey
	serializedSig := make([]byte, 1+len(signature)+len(publicKey))
	serializedSig[0] = 0x00 // Ed25519 flag
	copy(serializedSig[1:], signature)
	copy(serializedSig[1+len(signature):], publicKey)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(serializedSig)

	return []string{encoded}, nil
}

func (s *testSigner) GetAddress() (string, error) {
	publicKey := s.privateKey.Public().(ed25519.PublicKey)

	// For Ed25519, the signature scheme is 0x00
	const signatureScheme = 0x00

	// Create the data to hash: signature scheme byte || public key
	data := append([]byte{signatureScheme}, publicKey...)

	// Hash using Blake2b-256
	hash := blake2b.Sum256(data)

	// The Sui address is the hex representation of the hash
	return "0x" + hex.EncodeToString(hash[:]), nil
}

// fundTestAccount requests SUI tokens from the devnet faucet with retries
func (s *testSigner) fundTestAccount() error {
	address, err := s.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	// Use local faucet endpoint
	faucetEndpoints := []string{
		"http://localhost:9123/v2/gas",
	}

	for attempt := 1; attempt <= 3; attempt++ {
		for _, faucetURL := range faucetEndpoints {
			jsonBody := fmt.Sprintf(`{"FixedAmountRequest": {"recipient": "%s"}}`, address)

			client := &http.Client{Timeout: 30 * time.Second}
			req, err := http.NewRequest(http.MethodPost, faucetURL, strings.NewReader(jsonBody))
			if err != nil {
				continue
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Attempt %d - Network error for %s: %v\n", attempt, faucetURL, err)
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
				fmt.Printf("Successfully funded account %s from %s\n", address, faucetURL)
				// Wait for transaction to be processed
				time.Sleep(5 * time.Second)
				return nil
			}

			fmt.Printf("Attempt %d - Faucet %s returned status %d: %s\n", attempt, faucetURL, resp.StatusCode, string(body))
		}

		if attempt < 3 {
			fmt.Printf("Waiting 10 seconds before retry attempt %d...\n", attempt+1)
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("Sui devnet faucet is currently experiencing service issues and cannot allocate gas coins. This is a known temporary issue with the public devnet. Address: %s", address)
}

func (s *SuiTestSuite) SetupSuite() {
	s.client = sui.NewSuiClient("http://localhost:9000")

	testSigner, err := newTestSigner()
	s.Require().NoError(err, "Failed to create test signer")

	err = testSigner.fundTestAccount()
	s.Require().NoError(err, "Failed to fund test account")

	s.signer = testSigner
	s.chainSelector = types.ChainSelector(18395503381733958356)
}
**/

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

func (s *SuiTestSuite) DeployMCMSUserContract() {
	gasBudget := uint64(300_000_000)
	signerAddress, err := s.signer.GetAddress()
	s.Require().NoError(err, "Failed to get address")

	mcmsUserPackage, tx, err := mcms_user.PublishMCMSUser(context.Background(), &bind.CallOpts{
		Signer:           s.signer,
		GasBudget:        &gasBudget,
		WaitForExecution: true,
	}, s.client, s.mcmsPackageId, signerAddress)
	s.Require().NoError(err, "Failed to publish MCMS user package")

	s.mcmsUserPackageId = mcmsUserPackage.Address()
	s.mcmsUser = mcmsUserPackage.MCMSUser()

	userDataObj, err1 := bind.FindObjectIdFromPublishTx(*tx, "mcms_user", "UserData")
	mcmsUserOwnerCapObj, err2 := bind.FindObjectIdFromPublishTx(*tx, "mcms_user", "OwnerCap")

	if err1 != nil || err2 != nil {
		s.T().Fatalf("Failed to find object IDs in publish tx: %v, %v", err1, err2)
	}

	s.mcmsUserOwnerCapObj = mcmsUserOwnerCapObj
	s.stateObj = userDataObj
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
