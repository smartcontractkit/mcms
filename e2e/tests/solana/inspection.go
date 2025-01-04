//go:build e2e
// +build e2e

package e2e_solana

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
)

func (s *SolanaTestSuite) Test_Solana_InitializeMCM() {
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
		uint64(s.chainSelector),
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

	require.Equal(s.T(), uint64(s.chainSelector), configAccount.ChainId)
	require.Equal(s.T(), wallet.PublicKey(), configAccount.Owner)
}
