//go:build e2e

package aptos

import (
	"os"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"

	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
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
	testnet := false
	if testnet {
		// TODO remove this after testing
		a.TestSetup.AptosRPCClient, _ = aptos.NewNodeClient("https://api.testnet.aptoslabs.com/v1", 2)
		a.ChainSelector = types.ChainSelector(cselectors.APTOS_TESTNET.Selector)
		deployerKey := &crypto.Ed25519PrivateKey{}
		err := godotenv.Load("../custom_configs/.env")
		a.Require().NoError(err)
		userKey := os.Getenv("USER_KEY")
		err = deployerKey.FromHex(userKey)
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
	mcmsSeed := mcms.DefaultSeed + time.Now().String()
	addr, tx, mcmsContract, err := mcms.DeployToResourceAccount(a.deployerAccount, a.TestSetup.AptosRPCClient, mcmsSeed)
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
		metadata               = "0x0a546573744d6f64756c650100000000000000004030314531344334324233453535384342303434393742434643394537383646313130334431443446374630384434413438464633303541344531443432323932ac021f8b08000000000002ff7590bd6e83301485773f85c5c2d2802124692a75a82a7563eb861032f6052cc046b621a9a2bc7b6d42dba9db3d57e7bb3fa79828eb690b25927404fc8a834f3036577c1e20400b682394f4ed24221109109d6da7b4719da244a8a09c6b30064c8946369aca3ad8bbab60d55b890a0ecbee3f2fb92684fcf9bd4c1ecc04928364c2236f9355e643bb232f4af7ce78c3ad58f9cedac9bcc4b193dd5c474c8d31f5e6dd406bb3954c69889c2178c21a164fed33921ed2f404f5617f3cb12c3bd7cd9953d8339ea6c7060e2c793e9d49e60833d75c680f3d868d6a81b8f93965dbf0ab037c47ef1d157210b2cfa9fc5a453e0f5618d1ae870f8ad1c1cf8ba2d8bfed76f8702ab7688de486c335217591a043d708c99584f8ee26a36f50a780f5b101000001096d636d735f75736572cc061f8b08000000000002ffd556dd6adb3014becf539ce522d89b699b3046507e28741b8cb1f4a22dbb1843c8b6927ab5ad4c929b6625efbe23dbb165c7c9606317358458e7e8fcebfbe4448459cc2109124535579a90fc35535cc2730ff0c157503a24844b29e4a429136b1d89b425545a46e98a90e71b1e2f3db8c997bbf6a66895f20377fe0f1e600ed7f9ffa457694d5284f881a2e89cb364d2d2e4494bbe8a50bd2ded02912a0d1fee169f17d75f17f4e3dde2eaf6d3f58240f6ee2dcc60586e438b2cd0708715bf679ac13d53f0c0b71e8452ac3d540bc9cb5698274a1f45c04cd50a3d8dbd4ac14859692df2093c621d424eb3f1bc16070458184aae542d0bd1d97034ced7bb22b165969a5f606251917287c9d5b00a02b81ad9fe5d60c1cf2c42b7752d75da31d7a65d9286463e035fe0343774150b9fc534c9f4746f34772eabd3e04e2a0795f199d50174d42d7f63da7b686a229b2aba747ea11b4d8ef6406f44d9837dfbca2698cebd84f28313e587c7ca8fd248d32487a9b3cefc3852f75c1218140072ad2299426ffa9553680829bb44c5b23674613683ba420f866e8903f324e291532d9c4a609ecad66b883bbadc89910bef408f50d97344a69763c7eff7ddc35d157abe7d3f5422862e2f9e3a7c23902e1ac25db56a546a130621c51bce82a7b85e8b28d5c79ad04ebde24bac016e58b28ef917145db138f659f000cf8d04ace196c4d3616228c8b08fb1ed55d183fc38e4e1ea2ca7b7c4b0d5dca109d7cc9c2402057b4e6fe72e8182a09151f3ffa941cafc2452cecf612924b0740b01e683d78058b683027f0af85ae7620f565cd3a04c9dae996489828dc8e2b0e19345313a0d41df73d092a58ae5a02e76024340eab3065c1d5a9c63af2200a464cc154f706332ad497664e3d8e7bd7342f6c930b12bcaf1b76885c87cbd1f7a2e7006fb0d164318bbe27232fc52dd5484a47ce3e4895b41a22538ed2068d5b7f9beefb6a0652218fa68f90f394e316271f48bd3224d67808c5626636568f9189df051808e66e3136e6cdb827668a4a8b971b38487cea0d3e8e0322bf8dbdab5031ee39dfec7eee04df037dd2909f1dfda6320f4df3ab3bfe28e75a659728e1ac8bfcb0841d2c504438a861809f9ebe0cbc7adbd594771cf10a9998a5b12d4aef71b9a8a237c190a000000000400000000000000000000000000000000000000000000000000000000000000010e4170746f734672616d65776f726b00000000000000000000000000000000000000000000000000000000000000010b4170746f735374646c696200000000000000000000000000000000000000000000000000000000000000010a4d6f76655374646c6962abababababababababababababababababababababababababababababababab1a436861696e6c696e6b4d616e79436861696e4d756c746973696700"
		placeholderMCMSUser    = "EFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEFEF"
		placeholderMCMSAddress = "ABABABABABABABABABABABABABABABABABABABABABABABABABABABABABABABAB"
	)
	var (
		bytecode = []string{
			"0xa11ceb0b0700000a0c01001002101c032c6a04960106059c0171078d028f03089c056006fc057510f106510ac207140cd6078f020de5090a00000107010c020f01150117021a01230001020000030e0001060700041407010001051607010000061c0200000b00010001020d00020001010e030400010310070201020100110801000100120a01000100130c0d01080103180e0f010201011910110001061b03120001061d13040001061e13030001061f1401000105200116010001062113020001062213150001072417170001030607060d1501060c000105010a02010802010201080003060c080209000208020a0203070801070802070a020205040307080107050704010b03010900010b04010402050900030c08020a020106080201060a0201080501070805010608050104010b040109000103070a0208020a02080508020405096d636d735f757365721253616d706c654d636d7343616c6c6261636b0b64756d6d795f6669656c640855736572446174610b696e766f636174696f6e73016106537472696e6706737472696e670162016301640b696e69745f6d6f64756c65067369676e65720a616464726573735f6f6604757466380d6d636d735f72656769737472791372656769737465725f656e747279706f696e740c66756e6374696f6e5f6f6e650c66756e6374696f6e5f74776f0f6d636d735f656e747279706f696e74064f626a656374066f626a656374064f7074696f6e066f7074696f6e136765745f63616c6c6261636b5f706172616d730562797465730a6263735f73747265616d036e65770942435353747265616d12646573657269616c697a655f737472696e6715646573657269616c697a655f766563746f725f7538126173736572745f69735f636f6e73756d6564046e6f6e6513646573657269616c697a655f6164647265737310646573657269616c697a655f75313238056572726f7210696e76616c69645f617267756d656e74efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef0000000000000000000000000000000000000000000000000000000000000001abababababababababababababababababababababababababababababababab0520efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef0a020100052000000000000000000000000000000000000000000000000000000000000000000a020a096d636d735f757365720a020d0c66756e6374696f6e5f6f6e650a020d0c66756e6374696f6e5f74776f14636f6d70696c6174696f6e5f6d65746164617461090003322e3003322e31126170746f733a3a6d657461646174615f76311e0101000000000000001145554e4b4e4f574e5f46554e4354494f4e00000000020102010102050402050802080a0209050a0400000000011a0a00110107002104160a00310007011102400500000000000000000702320000000000000000000000000000000012012d010b0007031102091200380001020b0001060100000000000000270400000101091807002a010c020a021000143101160a020f00150a020f010c030b000b03150b020f020c040b010b04150205000001010b1807002a010c020a021000143101160a020f00150a020f030c030b000b03150b020f040c040b010b0415020601000101182a070009120038010c010c02010e021108140c030b0111090c040a03070421041b0d04110a0d04110b0e04110c11043802020b0307052104270d04110e0d04110f0e04110c110505190601000000000000001110270100010101020103010400",
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
