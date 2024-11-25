//go:build e2e
// +build e2e

package e2e

import (
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

// SigningTestSuite tests signing a proposal and converting back to a file
type SigningTestSuite struct {
	suite.Suite
	TestSetup

	client          *ethclient.Client
	contractAddress string
	deployerKey     common.Address
	signerAddresses []common.Address
	auth            *bind.TransactOpts
	selector        types.ChainSelector
}

// SetupSuite runs before the test suite
func (s *SigningTestSuite) SetupSuite() {
	in, err := framework.Load[Config](s.T())
	require.NoError(s.T(), err, "Failed to load configuration")
	// Load the proposal from a file
	// Create the file
	file, err := os.Create("proposal-testing.json")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	// convert chain ID string from config to number
	chainIDNum, ok := new(big.Int).SetString(in.BlockchainA.ChainID, 10)
	require.True(s.T(), ok, "Failed to parse chain ID")

	chainSelector, err := chain_selectors.SelectorFromChainId(chainIDNum.Uint64())
	require.NoError(s.T(), err, "Failed to get chain selector from chain ID")
	s.selector = types.ChainSelector(chainSelector)

}

func (s SigningTestSuite) TestReadAndSign() {
	// Read the proposal from the file
	file, err := os.Open("proposal-testing.json")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	proposal, err := mcms.NewProposal(file)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), proposal)
	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.selector: evm.NewInspector(s.client),
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	require.NoError(s.T(), err)
	signature, err := signable.SignAndAppend(
		mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])),
	)
	s.Require().NoError(err)
	s.Require().NotNil(signature)
}
