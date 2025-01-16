//go:build e2e
// +build e2e

package solanae2e

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

var testPDASeedInspect = [32]byte{'t', 'e', 's', 't', '-', 'i', 'n', 's', 'p', 'e', 'c', 't'}

// InspectSolanaTestSuite is the test suite for the Inspection functionality
type InspectSolanaTestSuite struct {
	*SolanaTestSuite
}

// SetupSuite runs before the test suite
func (s *InspectSolanaTestSuite) SetupSuite() {
	s.SolanaTestSuite.SetupSuite()
	s.SetupMCM(testPDASeedInspect)
}

// TestGetOpCount tests the GetOpCount functionality
func (s *InspectSolanaTestSuite) TestGetOpCount() {
	ctx := context.Background()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	opCount, err := inspector.GetOpCount(ctx, solanasdk.ContractAddress(s.MCMProgramID, testPDASeedInspect))

	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count does not match")
}

// TestGetRoot tests the GetRoot functionality
func (s *InspectSolanaTestSuite) TestGetRoot() {
	ctx := context.Background()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	root, validUntil, err := inspector.GetRoot(ctx, solanasdk.ContractAddress(s.MCMProgramID, testPDASeedInspect))

	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(ethcommon.Hash{}, root, "Roots do not match")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil does not match")
}

// TestGetRootMetadata tests the GetRootMetadata functionality
func (s *InspectSolanaTestSuite) TestGetRootMetadata() {
	ctx := context.Background()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	address := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedInspect)
	metadata, err := inspector.GetRootMetadata(ctx, address)

	s.Require().NoError(err, "Failed to get root metadata from contract")
	s.Require().Equal(address, metadata.MCMAddress, "MCMAddress does not match")
	s.Require().Equal(uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
