//go:build e2e

package aptos

import (
	"os"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"github.com/stretchr/testify/suite"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	aptosutil "github.com/smartcontractkit/mcms/e2e/utils/aptos"
	"github.com/smartcontractkit/mcms/types"
)

type AptosTestSuite struct {
	suite.Suite
	e2e.TestSetup

	deployerAccount aptos.TransactionSigner

	ChainSelector    types.ChainSelector
	MCMContract      *mcms.MCMS
	MCMSUserContract aptos.AccountAddress
}

func (a *AptosTestSuite) SetupSuite() {
	testnet := true
	if testnet {
		// TODO remove this after testing
		a.TestSetup.AptosRPCClient, _ = aptos.NewNodeClient("https://api.testnet.aptoslabs.com/v1", 2)
		a.ChainSelector = types.ChainSelector(cselectors.APTOS_TESTNET.Selector)
		deployerKey := &crypto.Ed25519PrivateKey{}
		err := deployerKey.FromHex(os.Getenv("USER_KEY"))
		a.Require().NoError(err)
		a.deployerAccount, err = aptos.NewAccountFromSigner(deployerKey)
		a.Require().NoError(err)
	} else {
		a.TestSetup = *e2e.InitializeSharedTestSetup(a.T())
		details, err := cselectors.GetChainDetailsByChainIDAndFamily(a.AptosChain.ChainID, cselectors.FamilyAptos)
		a.Require().NoError(err)
		a.ChainSelector = types.ChainSelector(details.ChainSelector)

		// Set up deployer account, it's automatically funded by CTF when setting up the Aptos chain
		// Instead of using (*Ed25519PrivateKey).FromHex directly, parse manually to pass the strict=false flag
		bytes, err := crypto.ParsePrivateKey(blockchain.DefaultAptosPrivateKey, crypto.PrivateKeyVariantEd25519, false)
		a.Require().NoError(err)
		deployerKey := &crypto.Ed25519PrivateKey{}
		err = deployerKey.FromBytes(bytes)
		a.Require().NoError(err)
		a.deployerAccount, err = aptos.NewAccountFromSigner(deployerKey)
		a.Require().NoError(err)
	}
}

func (a *AptosTestSuite) deployMCM() {
	addr, tx, mcmsContract, err := mcms.Deploy(a.deployerAccount, a.TestSetup.AptosRPCClient)
	a.Require().NoError(err)
	data, err := a.TestSetup.AptosRPCClient.WaitForTransaction(tx.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success)
	a.T().Logf("📃 Deployed MCM contract at %v in tx %v", addr.StringLong(), data.Hash)
	a.MCMContract = &mcmsContract
}

func (a *AptosTestSuite) deployMCMUser() {
	if a.MCMContract == nil {
		a.T().Fatal("MCMS contract not found. Can only deploy MCMS user contract after MCMS contract has been deployed.")
	}
	const (
		metadata               = "0x0a546573744d6f64756c65010000000000000000403937303630443931314330394231414534333433433430323736374536434145303532323735364244434343393743383643354438414245444644463645344685021f8b08000000000002ffa590cd4ac5301085f7798a92b537ed7529b810f1eeba73574a4993b10dcd4fc9a4bd8af8ee66daaa20b8926ce61ccef92649334b35c9015ae6a583e2bee0cf80a90e7ab1c0d90a114df0649f45252acee492c610313b4dcb5823b58e8008d832a71c762997295dbd3e5dfe77f8062456b78f5db87a8887c11a0debe9efede7aafa01903cff869079bb8366f01abc32c4799853c04bcc7f710d71cac1f762301b744c69c6bbb2cc725c7aa1822b25854f56f6788c2a441039c06f8a082bb59c34de0319b8f4da6cabf7ac0b2b942f5f9b0ec0b7e6c5077b1c73d91a3fd5d2bf6da25e6c326886ed5e362869892744496fa3ca27a85657a9cf01000001096d636d735f75736572ab061f8b08000000000002ffcd565b6bdb30147eefaf38eb43b137b336618ca05c28741b8cb1f4a12d7b1843c8b2927ab52d4f92936621ff7d47b67c89d3b48cbd2c10121f9df39d8bcef725a98c8a4440ca534d8dd08690f26ba18582ed09e00bbf823611214229a9c63ddb4a64a66793b98965d6336aa3e26c49c8f646248b006ecac75ddf295e66a29f42863f05c7baaecbcff149736a0b75e546b1ce99e1f7dde0ea38e49a626ec15217c965a60d7cbc9b7f995f7f9bd34f77f3abdbcfd77302c5fb7730858173c398821bb8c3397c6086c13dd3f0203601444ae6011e4b25dc8000e26c2539b35d6bc41905cecc88ebb336840456d88354936234ab8d9c008b2225b4ae2d11820c860e6657d5b32832fbe636099599f0985a0e1a7cc0a76117dc07c67f1531a2b62dd4d526c2d801291a59eb14428917bba6cb44862ca169612675c8ccbb6c16c31fbbf026f46da76d8479dafec64eb41f68b3dafa0f4fc2ea64383edabb594bd77b3d35d7bc1dd9ffdd363fda7674aced388b0d4d4b8e7a791126b1c61d27705631c56f9a631ab1cc2bafb213e26643e5a20df3613a85b6b30006be5b768054ae0435b2750e0e07d8dbf38ba0b1e3a2d7fc2ecc62e485a7a77e7bda6cfdf71fad11b7fef2e2b183814b7fe11e769dbaf6d94d8812cb581ba1ba95f67237028645c00d4bf3447c45d3154b9290f187edcedf1bb363faa163c9794b77d83ad732272f2fa64c82daa736b98c3333b925561e661e4d8561f64e09548235b99df9042a4d44112b3f27765767cfec2ac0f9392ca402966d80633da8c672d14f0ae2918bdc94e60096c250ee4aa739532cd5b0964512ed61b23841d008ccbd00a358a65949abca131892c2bc6d022c65bc9a79287d58222ed1e1a53c91da3b32f83dec86d4e1063712915fd757591abcb3daa16161155709bae570a3ee846462ed95157692c48bb6fe3a09469d7695f4d4ef8cbdc2b724eda14702ef286649fc5bd0aa48ef0c35c395d2a9af41183e8350f18116a3a320075a5fc95cc7670722c19fb9175b44c1fcfb169d7efc4b8f76c75f6eaf96f363ed75eb2eb713cabf2184a016619e886258912227bc839f74bfc5eaac44cdc4cc0ed67742b03bf9035852f1060809000000000400000000000000000000000000000000000000000000000000000000000000010e4170746f734672616d65776f726b00000000000000000000000000000000000000000000000000000000000000010b4170746f735374646c696200000000000000000000000000000000000000000000000000000000000000010a4d6f76655374646c6962abababababababababababababababababababababababababababababababab1a436861696e6c696e6b4d616e79436861696e4d756c746973696700"
		placeholderMCMSUser    = "EFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEF"
		placeholderMCMSAddress = "ABABABABABABABABABABABABABABABABABABABABABABABABABABABABABABABAB"
	)
	var (
		bytecode = []string{
			"0xa11ceb0b0700000a0c01001002101c032c64049001060596016207f801f30208eb046006cb057510c006510a9107140ca50783020da8090a00000107010c020f01150117021a01220001020000030e0001060700041407010001051607010000061c0200000b00010001020d00020001010e030400010310070101020100110801000100120a01000100130c0d01080103180e0801020101190f100001061b03110001061d12040001061e12030001051f0114010001062012020001062112130001072315150001030607060c1301060c000105010a02010802010201080003060c080209000208020a0203070801070802070a020205040307080107050704010b03010900010b0401040109000106080201060a02010805010708050104010b040109000103040a0208020a020805096d636d735f757365721253616d706c654d636d7343616c6c6261636b0b64756d6d795f6669656c640855736572446174610b696e766f636174696f6e73016106537472696e6706737472696e670162016301640b696e69745f6d6f64756c65067369676e65720a616464726573735f6f6604757466380f6d636d735f646973706174636865720872656769737465720c66756e6374696f6e5f6f6e650c66756e6374696f6e5f74776f0f6d636d735f656e747279706f696e74064f626a656374066f626a656374064f7074696f6e066f7074696f6e136765745f63616c6c6261636b5f706172616d730562797465730a6263735f73747265616d036e65770942435353747265616d12646573657269616c697a655f737472696e6715646573657269616c697a655f766563746f725f7538046e6f6e6513646573657269616c697a655f6164647265737310646573657269616c697a655f75313238056572726f7210696e76616c69645f617267756d656e74efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef0000000000000000000000000000000000000000000000000000000000000001abababababababababababababababababababababababababababababababab0520efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef0a020100052000000000000000000000000000000000000000000000000000000000000000000a020a096d636d735f757365720a020d0c66756e6374696f6e5f6f6e650a020d0c66756e6374696f6e5f74776f14636f6d70696c6174696f6e5f6d65746164617461090003322e3003322e31126170746f733a3a6d657461646174615f76311e0101000000000000001145554e4b4e4f574e5f46554e4354494f4e00000000020102010102050402050802080a0209050a040000000001190a00110107002104150a00310007011102400500000000000000000702320000000000000000000000000000000012012d010b00070311020912003800020b0001060100000000000000270400000101091807002a010c020a021000143101160a020f00150a020f010c030b000b03150b020f020c040b010b04150205000001010b1807002a010c020a021000143101160a020f00150a020f030c030b000b03150b020f040c040b010b0415020601000101162409120038010c010c020e021108140c030b0111090c040a0307042104170d04110a0d04110b11043802020b0307052104210d04110d0d04110e11050515060100000000000000110f270100010101020103010400",
		}
	)
	// Pre-calculate named object address
	objectAddress, err := aptosutil.NextObjectCodeDeploymentAddress(a.TestSetup.AptosRPCClient, a.deployerAccount.AccountAddress())
	a.Require().NoError(err)

	payload, err := aptosutil.ObjectCodeDeploymentPublish(metadata, bytecode, map[string]string{
		placeholderMCMSUser:    objectAddress.StringLong(),
		placeholderMCMSAddress: a.MCMContract.Address.StringLong(),
	})
	a.Require().NoError(err)
	data, err := aptosutil.BuildSignSubmitAndWaitForTransaction(a.TestSetup.AptosRPCClient, a.deployerAccount, payload)
	a.Require().NoError(err)

	// Look for event that contains the newly deployed object
	for _, event := range data.Events {
		if event.Type == "0x1::object_code_deployment::Publish" {
			if address, ok := event.Data["object_address"]; ok {
				a.Require().NoError(a.MCMSUserContract.ParseStringRelaxed(address.(string)))
			}
		}
	}
	a.T().Logf("📃 Deployed MCMUser contract at %v in tx %v", a.MCMContract.Address.StringLong(), data.Hash)
}
