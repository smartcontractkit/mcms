package e2e

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/gagliardetto/solana-go"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/suite"

	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// SetConfigTestSuite defines the test suite for setting config using  solana mcms program.
type SetConfigTestSuite struct {
	suite.Suite
	chainSelector         mcmtypes.ChainSelector
	deployerPrivateKey    solana.PrivateKey
	mcmsProgramPublicKey  *solana.PublicKey
	mcmsProgramPrivateKey solana.PrivateKey
	TestSetup
}

// SetupSuite runs before the test suite
func (s *SetConfigTestSuite) SetupSuite() {
	ctx := context.Background()
	s.TestSetup = *InitializeSharedTestSetup(s.T())

	// Create deployer account with funds
	payer, err := CreateFundedTestAccount(ctx, s.ClientSolana)
	s.Require().NoError(err)
	s.deployerPrivateKey = payer
	// Create MCMS Program account
	programKey, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	programPublicKey := programKey.PublicKey()
	s.mcmsProgramPublicKey = &programPublicKey
	s.mcmsProgramPrivateKey = programKey
	// Parse ChainID from string to int64
	_, ok := new(big.Int).SetString(s.BlockchainA.Out.ChainID, 10)
	s.Require().True(ok, "Failed to parse chain ID")

	// Read program binary
	programData, err := os.ReadFile(MCMSBinPath)
	if err != nil {
		log.Fatalf("Failed to read program binary: %v", err)
	}

	// Deploy MCMS program
	err = DeployProgramUsingBpfLoader(ctx, s.ClientSolana, s.ClientWSSolana, &payer, &programKey, programData)
	s.Require().NoError(err)

	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.BlockchainA.Out.Family)
	s.Require().NoError(err)
	s.chainSelector = mcmtypes.ChainSelector(chainDetails.ChainSelector)
}
func (s *SetConfigTestSuite) TestSetConfig() {
	// Test SetConfig
	fmt.Println("Testing SetConfig...")
}
