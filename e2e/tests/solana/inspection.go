//go:build e2e
// +build e2e

package solanae2e

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

var testPDASeedGetOpCount = [32]byte{'t', 'e', 's', 't', '-', 'g', 'e', 't', 'o', 'p'}
var testPDASeedGetRoot = [32]byte{'t', 'e', 's', 't', '-', 'g', 'e', 't', 'r', 'o', 'o', 't'}
var testPDASeedGetRootMeta = [32]byte{'t', 'e', 's', 't', '-', 'g', 'e', 't', 'm', 'e', 't', 'a'}

// TestGetOpCount tests the GetOpCount functionality
func (s *SolanaTestSuite) TestGetOpCount() {
	ctx := context.Background()
	InitializeMCMProgram(ctx, s.T(), s.SolanaClient, s.MCMProgramID, testPDASeedGetOpCount, uint64(s.ChainSelector))

	inspector := solanasdk.NewInspector(s.SolanaClient)
	opCount, err := inspector.GetOpCount(ctx, solanasdk.ContractAddress(s.MCMProgramID, testPDASeedGetOpCount))

	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count does not match")
}

// TestGetRoot tests the GetRoot functionality
func (s *SolanaTestSuite) TestGetRoot() {
	ctx := context.Background()
	InitializeMCMProgram(ctx, s.T(), s.SolanaClient, s.MCMProgramID, testPDASeedGetRoot, uint64(s.ChainSelector))

	inspector := solanasdk.NewInspector(s.SolanaClient)
	root, validUntil, err := inspector.GetRoot(ctx, solanasdk.ContractAddress(s.MCMProgramID, testPDASeedGetRoot))

	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(ethcommon.Hash{}, root, "Roots do not match")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil does not match")
}

// TestGetRootMetadata tests the GetRootMetadata functionality
func (s *SolanaTestSuite) TestGetRootMetadata() {
	ctx := context.Background()
	InitializeMCMProgram(ctx, s.T(), s.SolanaClient, s.MCMProgramID, testPDASeedGetRootMeta, uint64(s.ChainSelector))

	inspector := solanasdk.NewInspector(s.SolanaClient)
	address := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedGetRootMeta)
	metadata, err := inspector.GetRootMetadata(ctx, address)

	s.Require().NoError(err, "Failed to get root metadata from contract")
	s.Require().Equal(address, metadata.MCMAddress, "MCMAddress does not match")
	s.Require().Equal(uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
