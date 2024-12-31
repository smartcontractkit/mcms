//go:build e2e
// +build e2e

package e2e

import (
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/mcms"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SolanaInspectionTestSuite struct {
	suite.Suite
	TestSetup
}

// this key matches the public key in the config.toml so it gets funded by the genesis block
var privateKey = "DmPfeHBC8Brf8s5qQXi25bmJ996v6BHRtaLc6AH51yFGSqQpUMy1oHkbbXobPNBdgGH2F29PAmoq9ZZua4K9vCc"

// SetupSuite runs before the test suite
func (s *SolanaInspectionTestSuite) SetupSuite() {
	s.TestSetup = *InitializeSharedTestSetup(s.T())
}

func (s *SolanaInspectionTestSuite) TestSetupMCM() {
	mcm.SetProgramID(config.McmProgram)

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	require.NoError(s.T(), err)

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
		wallet.PublicKey(),
		solana.SystemProgramID,
		config.McmProgram,
		programData.Address,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
	).ValidateAndBuild()
	require.NoError(s.T(), err)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, wallet, config.DefaultCommitment)

	// get config and validate
	var configAccount mcm.MultisigConfig
	err = common.GetAccountDataBorshInto(ctx, s.SolanaClient, multisigConfigPDA, config.DefaultCommitment, &configAccount)
	require.NoError(s.T(), err, "failed to get account data")

	require.Equal(s.T(), config.TestChainID, configAccount.ChainId)
	require.Equal(s.T(), wallet.PublicKey(), configAccount.Owner)
}
