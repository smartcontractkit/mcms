//go:build e2e

package sui

import (
	"context"
	"crypto/ecdsa"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

// InspectionTestSuite defines the test suite for Sui inspection tests
type InspectionTestSuite struct {
	SuiTestSuite
}

// SetupSuite runs before the test suite
func (s *InspectionTestSuite) SetupSuite() {
	s.SuiTestSuite.SetupSuite()
	s.SuiTestSuite.DeployMCMSContract()
}

// TestGetConfig checks contract configuration
func (s *InspectionTestSuite) TestGetConfig() {
	ctx := context.Background()

	// First, set up a configuration to test with
	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "Failed to create inspector")

	proposers := [3]common.Address{}
	proposerKeys := [3]*ecdsa.PrivateKey{}
	for i := range proposers {
		proposerKeys[i], _ = crypto.GenerateKey()
		proposers[i] = crypto.PubkeyToAddress(proposerKeys[i].PublicKey)
	}
	slices.SortFunc(proposers[:], func(a, b common.Address) int {
		return a.Cmp(b)
	})

	testConfig := &types.Config{
		Quorum:  2,
		Signers: proposers[:],
	}

	// Set the configuration using the configurer
	configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleProposer, s.mcmsPackageId, s.ownerCapObj, uint64(s.chainSelector))
	s.Require().NoError(err, "Failed to create configurer")

	configTx, err := configurer.SetConfig(ctx, s.mcmsObj, testConfig, true)
	s.Require().NoError(err, "Failed to set test configuration")
	s.T().Logf("✅ SetConfig in tx: %s", configTx.Hash)

	// Now test getting the configuration
	config, err := inspector.GetConfig(ctx, s.mcmsObj)
	s.Require().NoError(err, "Failed to get contract configuration")
	s.Require().NotNil(config, "Contract configuration is nil")

	// Verify the configuration matches what we set
	s.Require().Equal(testConfig.Quorum, config.Quorum, "Quorum should match")
	s.Require().Len(config.Signers, len(testConfig.Signers), "Number of signers should match")
	s.Require().Equal(proposers[0], config.Signers[0], "First signer should match")
	s.Require().Equal(proposers[1], config.Signers[1], "Second signer should match")

	// Verify group signers
	s.Require().Len(config.GroupSigners, len(testConfig.GroupSigners), "Number of group signers should match")
	if len(config.GroupSigners) > 0 {
		s.Require().Equal(testConfig.GroupSigners[0].Quorum, config.GroupSigners[0].Quorum, "Group quorum should match")
		s.Require().Equal(testConfig.GroupSigners[0].Signers[0], config.GroupSigners[0].Signers[0], "Group signer should match")
	}
}

// TestGetOpCount checks contract operation count
func (s *InspectionTestSuite) TestGetOpCount() {
	ctx := context.Background()

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "Failed to create inspector")

	opCount, err := inspector.GetOpCount(ctx, s.mcmsObj)
	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count should be 0 for a fresh contract")
}

// TestGetRoot checks contract root
func (s *InspectionTestSuite) TestGetRoot() {
	ctx := context.Background()

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "Failed to create inspector")

	root, validUntil, err := inspector.GetRoot(ctx, s.mcmsObj)
	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(common.Hash{}, root, "Root should be empty for a fresh contract")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil should be 0 for a fresh contract")
}

// TestGetRootMetadata checks contract root metadata
func (s *InspectionTestSuite) TestGetRootMetadata() {
	ctx := context.Background()

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageId, suisdk.TimelockRoleProposer)
	s.Require().NoError(err, "Failed to create inspector")

	metadata, err := inspector.GetRootMetadata(ctx, s.mcmsObj)
	s.Require().NoError(err, "Failed to get root metadata from contract")
	s.Require().Equal(uint64(0), metadata.StartingOpCount, "StartingOpCount should be 0 for a fresh contract")
	s.Require().NotEmpty(metadata.MCMAddress, "MCMAddress should not be empty")
}
