//go:build e2e

package aptos

import (
	"time"

	"github.com/stretchr/testify/suite"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"

	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	mcmstest "github.com/smartcontractkit/chainlink-aptos/bindings/mcms_test"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	"github.com/smartcontractkit/mcms/types"
)

type TestSuite struct {
	suite.Suite
	e2e.TestSetup

	deployerAccount aptos.TransactionSigner

	ChainSelector    types.ChainSelector
	MCMSContract     mcms.MCMS
	MCMSTestContract mcmstest.MCMSTest
}

func (a *TestSuite) SetupSuite() {
	a.TestSetup = *e2e.InitializeSharedTestSetup(a.T())
	details, err := chainsel.GetChainDetailsByChainIDAndFamily(a.AptosChain.ChainID, chainsel.FamilyAptos)
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

func (a *TestSuite) deployMCMSContract() {
	mcmsSeed := mcms.DefaultSeed + time.Now().String()
	addr, tx, mcmsContract, err := mcms.DeployToResourceAccount(a.deployerAccount, a.AptosRPCClient, mcmsSeed)
	a.Require().NoError(err)
	data, err := a.AptosRPCClient.WaitForTransaction(tx.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("📃 Deployed MCM contract at %v in tx %v", addr.StringLong(), data.Hash)
	a.MCMSContract = mcmsContract
}

func (a *TestSuite) deployMCMSTestContract() {
	if a.MCMSContract == nil {
		a.T().Fatal("MCMS contract not found. Can only deploy MCMS user contract after MCMS contract has been deployed.")
	}
	addr, tx, mcmsTestContract, err := mcmstest.DeployToObject(a.deployerAccount, a.AptosRPCClient, a.MCMSContract.Address())
	a.Require().NoError(err)
	data, err := a.AptosRPCClient.WaitForTransaction(tx.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
	a.T().Logf("📃 Deployed MCMS Test contract at %v in tx %v", addr.StringLong(), data.Hash)
	a.MCMSTestContract = mcmsTestContract
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
