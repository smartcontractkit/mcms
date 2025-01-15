//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

func (s *SolanaTestSuite) SetupMCM() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)

	bindings.SetProgramID(s.MCMProgramID)

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	configPDA, err := solanasdk.FindConfigPDA(s.MCMProgramID, testPDASeed)
	s.Require().NoError(err)
	rootMetadataPDA, err := solanasdk.FindRootMetadataPDA(s.MCMProgramID, testPDASeed)
	s.Require().NoError(err)
	expiringRootAndOpCountPDA, err := solanasdk.FindExpiringRootAndOpCountPDA(s.MCMProgramID, testPDASeed)
	s.Require().NoError(err)

	// get program data account
	data, err := s.SolanaClient.GetAccountInfoWithOpts(ctx, s.MCMProgramID, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	s.Require().NoError(err)

	// decode program data
	var programData struct {
		DataType uint32
		Address  solana.PublicKey
	}
	err = bin.UnmarshalBorsh(&programData, data.Bytes())
	s.Require().NoError(err)

	ix, err := bindings.NewInitializeInstruction(
		uint64(s.ChainSelector),
		testPDASeed,
		configPDA,
		wallet.PublicKey(),
		solana.SystemProgramID,
		s.MCMProgramID,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	).ValidateAndBuild()
	s.Require().NoError(err)

	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, wallet, rpc.CommitmentConfirmed)

	// get config and validate
	var configAccount bindings.MultisigConfig
	err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, configPDA, rpc.CommitmentConfirmed, &configAccount)
	s.Require().NoError(err, "failed to get account data")

	s.Require().Equal(uint64(s.ChainSelector), configAccount.ChainId)
	s.Require().Equal(wallet.PublicKey(), configAccount.Owner)
}

func (s *SolanaTestSuite) TestGetOpCount() {
	ctx := context.Background()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	opCount, err := inspector.GetOpCount(ctx, solanasdk.ContractAddress(s.MCMProgramID, testPDASeed))

	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count does not match")
}

func (s *SolanaTestSuite) TestGetRoot() {
	ctx := context.Background()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	root, validUntil, err := inspector.GetRoot(ctx, solanasdk.ContractAddress(s.MCMProgramID, testPDASeed))

	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(ethcommon.Hash{}, root, "Roots do not match")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil does not match")
}

func (s *SolanaTestSuite) TestGetRootMetadata() {
	ctx := context.Background()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	address := solanasdk.ContractAddress(s.MCMProgramID, testPDASeed)
	metadata, err := inspector.GetRootMetadata(ctx, address)

	s.Require().NoError(err, "Failed to get root metadata from contract")
	s.Require().Equal(address, metadata.MCMAddress, "MCMAddress does not match")
	s.Require().Equal(uint64(0), metadata.StartingOpCount, "StartingOpCount does not match")
}
