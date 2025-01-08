//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	cselectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

type SolanaInspectionTestSuite struct {
	suite.Suite
	TestSetup

	ChainSelector types.ChainSelector
	MCMProgramID  solana.PublicKey
}

var (
	// this key matches the public key in the config.toml so it gets funded by the genesis block
	privateKey  = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"
	testPDASeed = [32]byte{'t', 'e', 's', 't', '-', 'm', 'c', 'm'}
)

func (s *SolanaInspectionTestSuite) SetupSuite() {
	s.TestSetup = *InitializeSharedTestSetup(s.T())
	s.MCMProgramID = solana.MustPublicKeyFromBase58(s.SolanaChain.SolanaPrograms["mcm"])

	details, err := cselectors.GetChainDetailsByChainIDAndFamily(s.SolanaChain.ChainID, cselectors.FamilySolana)
	s.Require().NoError(err)
	s.ChainSelector = types.ChainSelector(details.ChainSelector)
}

func (s *SolanaInspectionTestSuite) TestSetupMCM() {
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
	require.NoError(s.T(), err)

	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, wallet, rpc.CommitmentConfirmed)

	// get config and validate
	var configAccount bindings.MultisigConfig
	err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, configPDA, rpc.CommitmentConfirmed, &configAccount)
	require.NoError(s.T(), err, "failed to get account data")

	require.Equal(s.T(), uint64(s.ChainSelector), configAccount.ChainId)
	require.Equal(s.T(), wallet.PublicKey(), configAccount.Owner)
}
