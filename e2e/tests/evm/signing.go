//go:build e2e
// +build e2e

package evme2e

import (
	"encoding/json"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/mcms"
	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// SigningTestSuite tests signing a proposal and converting back to a file
type SigningTestSuite struct {
	suite.Suite
	e2e.TestSetup

	client        *ethclient.Client
	chainSelector mcmtypes.ChainSelector
}

// SetupSuite runs before the test suite
func (s *SigningTestSuite) SetupSuite() {
	s.TestSetup = *e2e.InitializeSharedTestSetup(s.T())

	chainDetails, err := cselectors.GetChainDetailsByChainIDAndFamily(s.BlockchainA.Out.ChainID, s.BlockchainA.Out.Family)
	s.Require().NoError(err)
	s.chainSelector = mcmtypes.ChainSelector(chainDetails.ChainSelector)
}

func (s *SigningTestSuite) TestReadAndSign() {
	file, err := testutils.ReadFixture("proposal-testing.json")
	s.Require().NoError(err, "Failed to read fixture") // Check immediately after ReadFixture
	defer func(file *os.File) {
		if file != nil {
			err = file.Close()
			s.Require().NoError(err, "Failed to close file")
		}
	}(file)
	s.Require().NoError(err)
	proposal, err := mcms.NewProposal(file)
	s.Require().NoError(err)
	s.Require().NotNil(proposal)
	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.chainSelector: evm.NewInspector(s.client),
	}
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
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
		err = os.Remove(name)
		s.Require().NoError(err, "Failed to remove temp file")
	}(tmpFile.Name())
	err = mcms.WriteProposal(tmpFile, proposal)
	s.Require().NoError(err)

	// Read back the written proposal
	_, err = tmpFile.Seek(0, io.SeekStart)
	s.Require().NoError(err, "Failed to reset file pointer to the start")

	writtenProposal, err := mcms.NewProposal(tmpFile)
	s.Require().NoError(err)

	// Validate the appended signature
	signedProposalJSON, err := json.Marshal(writtenProposal)
	s.Require().NoError(err)

	var parsedProposal map[string]any
	err = json.Unmarshal(signedProposalJSON, &parsedProposal)
	s.Require().NoError(err)

	// Ensure the signature is present and matches
	signatures, ok := parsedProposal["signatures"].([]any)
	s.Require().True(ok, "Signatures field is missing or of the wrong type")
	s.Require().NotEmpty(signatures, "Signatures field is empty")

	// Verify the appended signature matches the expected value
	appendedSignature := signatures[len(signatures)-1].(map[string]any)
	s.Require().Equal(expected.R.Hex(), appendedSignature["R"])
	s.Require().Equal(expected.S.Hex(), appendedSignature["S"])
	s.Require().InDelta(expected.V, appendedSignature["V"], 1e-9)
}
