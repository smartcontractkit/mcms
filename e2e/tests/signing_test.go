//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	mcmtypes "github.com/smartcontractkit/mcms/types"
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
	chainSelector   mcmtypes.ChainSelector
}

// SetupSuite runs before the test suite
func (s *SigningTestSuite) SetupSuite() {
	s.TestSetup = *InitializeSharedTestSetup(s.T())

	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.Config.Settings.ChainFamily)
	s.Require().NoError(err)
	s.chainSelector = mcmtypes.ChainSelector(chainDetails.ChainSelector)
}

func (s *SigningTestSuite) TestReadAndSign() {
	file, err := testutils.ReadFixture("proposal-testing.json")
	defer file.Close()
	s.Require().NoError(err)
	proposal, err := mcms.NewProposal(file)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), proposal)
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelector: evm.NewInspector(s.client),
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	require.NoError(s.T(), err)
	signature, err := signable.SignAndAppend(
		mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])),
	)
	s.Require().NoError(err)
	expected := mcmtypes.Signature{
		R: common.HexToHash("0x51c12e8721bf27f35a0006b3e3ebd0dac111c4bb62dce7b0bd7a3475b2f708a5"),
		S: common.HexToHash("0x28f29f2a32f4cd9322883fa252742894cc2796a6fbe9cdabd0c6d996eed452f9"),
		V: 0,
	}
	s.Require().Equal(expected, signature)
	// Write the proposal back to a temp file
	tmpFile, err := os.CreateTemp("", "signed-proposal-*.json")
	s.Require().NoError(err)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(tmpFile.Name()) // Clean up the temp file after the test
	err = mcms.WriteProposal(tmpFile, proposal)
	s.Require().NoError(err)

	// Read back the written proposal
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return
	} // Reset file pointer to the start
	writtenProposal, err := mcms.NewProposal(tmpFile)
	s.Require().NoError(err)

	// Validate the appended signature
	signedProposalJSON, err := json.Marshal(writtenProposal)
	s.Require().NoError(err)

	var parsedProposal map[string]interface{}
	err = json.Unmarshal(signedProposalJSON, &parsedProposal)
	s.Require().NoError(err)

	// Ensure the signature is present and matches
	signatures, ok := parsedProposal["signatures"].([]interface{})
	s.Require().True(ok, "Signatures field is missing or of the wrong type")
	s.Require().NotEmpty(signatures, "Signatures field is empty")

	// Verify the appended signature matches the expected value
	appendedSignature := signatures[len(signatures)-1].(map[string]interface{})
	s.Require().Equal(expected.R.Hex(), appendedSignature["R"])
	s.Require().Equal(expected.S.Hex(), appendedSignature["S"])
	s.Require().Equal(float64(expected.V), appendedSignature["V"])
}
