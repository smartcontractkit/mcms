//go:build e2e
// +build e2e

package e2e

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/mcms"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
)

type SolanaInspectionTestSuite struct {
	suite.Suite
	TestSetup
	wallet solana.PrivateKey
}

// this key matches the public key in the config.toml so it gets funded by the genesis block
var privateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

func (s *SolanaInspectionTestSuite) SetupSuite() {
	s.TestSetup = *InitializeSharedTestSetup(s.T())

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	require.NoError(s.T(), err)
	s.wallet = wallet

	s.initializeMCM()
}

func (s *SolanaInspectionTestSuite) initializeMCM() {
	mcm.SetProgramID(config.McmProgram)

	ctx := tests.Context(s.T())

	seed := config.TestMsigNamePaddedBuffer
	multisigConfigPDA := mcms.McmConfigAddress(seed)
	rootMetadataPDA := mcms.RootMetadataAddress(seed)
	expiringRootAndOpCountPDA := mcms.ExpiringRootAndOpCountAddress(seed)

	// get program data account
	data, accErr := s.SolanaClient.GetAccountInfoWithOpts(ctx, config.McmProgram, &rpc.GetAccountInfoOpts{
		Commitment: config.DefaultCommitment,
	})
	require.NoError(s.T(), accErr)

	// decode program data
	var programData struct {
		DataType uint32
		Address  solana.PublicKey
	}
	require.NoError(s.T(), bin.UnmarshalBorsh(&programData, data.Bytes()))

	ix, err := mcm.NewInitializeInstruction(
		config.TestChainID,
		seed,
		multisigConfigPDA,
		s.wallet.PublicKey(),
		solana.SystemProgramID,
		config.McmProgram,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	).ValidateAndBuild()
	require.NoError(s.T(), err)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, s.wallet, config.DefaultCommitment)
}

func (s *SolanaInspectionTestSuite) TestGetOpCount() {
	ctx := context.TODO()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	contractID := solanasdk.NewSolanaContractID(config.McmProgram, config.TestMsigNamePaddedBuffer)

	opCount, err := inspector.GetOpCount(ctx, contractID)
	s.Require().NoError(err, "Failed to get op count")
	s.Require().Equal(uint64(0), opCount, "Operation count does not match")
}

func (s *SolanaInspectionTestSuite) TestGetRoot() {
	ctx := context.TODO()
	inspector := solanasdk.NewInspector(s.SolanaClient)
	contractID := solanasdk.NewSolanaContractID(config.McmProgram, config.TestMsigNamePaddedBuffer)

	root, validUntil, err := inspector.GetRoot(ctx, contractID)
	s.Require().NoError(err, "Failed to get root from contract")
	s.Require().Equal(common.Hash{}, root, "Roots do not match")
	s.Require().Equal(uint32(0), validUntil, "ValidUntil does not match")
}
