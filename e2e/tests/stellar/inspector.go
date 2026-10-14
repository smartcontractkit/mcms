package stellar

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
	stellarbindings "github.com/smartcontractkit/chainlink-stellar/bindings"
	stellardeployer "github.com/smartcontractkit/chainlink-stellar/deployment"
	"github.com/stretchr/testify/suite"

	e2e "github.com/smartcontractkit/mcms/e2e/tests"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// MCMSInspectorTestSuite exercises the read-side sdk.Inspector against a
// freshly deployed Stellar MCMS instance.
type MCMSInspectorTestSuite struct {
	suite.Suite
	e2e.TestSetup

	deployer      *stellardeployer.Deployer
	chainSelector mcmtypes.ChainSelector
	passphrase    string
	stellarSigner stellarbindings.Signer

	mcmsAddress string
	mcmsConfig  *mcmtypes.Config
	inspector   *stellarsdk.Inspector

	deploymentCounter uint64
}

func (s *MCMSInspectorTestSuite) SetupSuite() {
	env := bootstrapStellarChainEnv(s.T())

	s.TestSetup = env.testSetup
	s.stellarSigner = env.signer
	s.deployer = env.deployer
	s.chainSelector = env.chainSelector
	s.passphrase = env.passphrase

	// Group 0 contains signer 0.
	// Group 1 contains signer 1 and is a child of group 0.
	_, signerAddresses := generateSortedSigners(s.T(), 2)
	s.mcmsConfig = &mcmtypes.Config{
		Quorum:  1,
		Signers: []common.Address{signerAddresses[0]},
		GroupSigners: []mcmtypes.Config{
			{
				Quorum: 1,
				Signers: []common.Address{
					signerAddresses[1],
				},
			},
		},
	}

	s.mcmsAddress = deployStellarMCMSContract(
		s.T(),
		s.deployer,
		s.stellarSigner.Address(),
		s.chainSelector,
		s.mcmsConfig,
		fmt.Sprintf("e2e_insp_%d", s.nextDeploymentID()),
	)

	s.inspector = newStellarInspector(s.T(), s.StellarClient, s.stellarSigner, s.passphrase)
}

// TestGetConfig verifies the inspector reads back the config the contract was
// initialized with.
func (s *MCMSInspectorTestSuite) TestGetConfig() {
	ctx := s.T().Context()

	actualConfig, err := s.inspector.GetConfig(ctx, s.mcmsAddress)
	s.Require().NoError(err, "getting config from inspector")
	s.Require().NotNil(actualConfig, "config should not be nil")

	s.verifyConfigMatch(s.mcmsConfig, actualConfig)
}

// TestGetOpCount verifies a freshly initialized MCMS reports zero ops.
func (s *MCMSInspectorTestSuite) TestGetOpCount() {
	ctx := s.T().Context()

	opCount, err := s.inspector.GetOpCount(ctx, s.mcmsAddress)
	s.Require().NoError(err, "getting op count")
	s.Require().Equal(uint64(0), opCount, "initial op count should be 0")
}

// TestGetRoot verifies a freshly initialized MCMS reports the empty root.
func (s *MCMSInspectorTestSuite) TestGetRoot() {
	ctx := s.T().Context()

	root, validUntil, err := s.inspector.GetRoot(ctx, s.mcmsAddress)
	s.Require().NoError(err, "getting root")
	s.Require().Equal(common.Hash{}, root, "initial root should be empty")
	s.Require().Equal(uint32(0), validUntil, "initial root validity should be 0")
}

// TestGetRootMetadata verifies the documented fresh-contract behavior: the
// contract stores root metadata only when a root is set, so a freshly
// initialized MCMS returns MissingRootMetadata (contract error 52). The
// root-set path is covered by the execution suite.
func (s *MCMSInspectorTestSuite) TestGetRootMetadata() {
	ctx := s.T().Context()

	_, err := s.inspector.GetRootMetadata(ctx, s.mcmsAddress)
	s.Require().Error(err, "fresh MCMS has no root metadata")
	s.Require().Contains(err.Error(), "get_root_metadata")
}

// TestGetOwner verifies the deployer is the initial MCMS owner.
func (s *MCMSInspectorTestSuite) TestGetOwner() {
	ctx := s.T().Context()

	owner, err := s.inspector.GetOwner(ctx, s.mcmsAddress)
	s.Require().NoError(err, "getting owner")
	s.Require().NotNil(owner, "owner should not be nil")
	s.Require().Equal(s.stellarSigner.Address(), *owner, "deployer should own the MCMS")
}

// TestGetPendingOwner verifies no pending owner exists on a fresh MCMS.
func (s *MCMSInspectorTestSuite) TestGetPendingOwner() {
	ctx := s.T().Context()

	pendingOwner, err := s.inspector.GetPendingOwner(ctx, s.mcmsAddress)
	s.Require().NoError(err, "getting pending owner")
	s.Require().Nil(pendingOwner, "pending owner should be nil")
}

// TestGetChainNetworkID verifies the network identifier the contract was
// initialized with.
func (s *MCMSInspectorTestSuite) TestGetChainNetworkID() {
	ctx := s.T().Context()

	networkIDHex, err := chainsel.StellarChainIdFromSelector(uint64(s.chainSelector))
	s.Require().NoError(err)

	networkID, err := s.inspector.GetChainNetworkID(ctx, s.mcmsAddress)
	s.Require().NoError(err, "getting chain network ID")
	s.Require().Equal(common.HexToHash(networkIDHex), common.Hash(networkID), "network ID mismatch")
}

func (s *MCMSInspectorTestSuite) verifyConfigMatch(expected, actual *mcmtypes.Config) {
	s.T().Helper()

	s.Require().Equal(expected.Quorum, actual.Quorum, "quorum should match")
	s.Require().Len(actual.Signers, len(expected.Signers), "number of signers should match")

	for i, expectedSigner := range expected.Signers {
		s.Require().Equal(expectedSigner, actual.Signers[i], "signer %d should match", i)
	}

	s.Require().Len(actual.GroupSigners, len(expected.GroupSigners), "number of group signers should match")
	for i, expectedGroup := range expected.GroupSigners {
		s.verifyConfigMatch(&expectedGroup, &actual.GroupSigners[i])
	}
}

func (s *MCMSInspectorTestSuite) nextDeploymentID() uint64 {
	s.deploymentCounter++

	return s.deploymentCounter
}
