//go:build e2e

package solanae2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"

	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/external_program_cpi_stub"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"

	"github.com/smartcontractkit/mcms"
	chainwrappermocks "github.com/smartcontractkit/mcms/chainwrappers/mocks"
	e2ecommon "github.com/smartcontractkit/mcms/e2e/tests/common"
	e2esolanautils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedTimelockScheduleCancel = [32]byte{'t', 'e', 's', 't', '-', 's', 'c', 'h', 'e', 'd', 'c', 'a', 'n', 'c', 'e', 'l'}

func (s *TestSuite) TestScheduleAndCancelProposal() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.T().Cleanup(cancel)

	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	s.SetupMCM(testPDASeedTimelockScheduleCancel)
	s.SetupTimelock(testPDASeedTimelockScheduleCancel, 5*time.Minute)

	mcmAddress := solanasdk.ContractAddress(s.MCMProgramID, testPDASeedTimelockScheduleCancel)
	timelockAddress := solanasdk.ContractAddress(s.TimelockProgramID, testPDASeedTimelockScheduleCancel)

	mcmSignerPDA, err := solanasdk.FindSignerPDA(s.MCMProgramID, testPDASeedTimelockScheduleCancel)
	s.Require().NoError(err)
	timelockSignerPDA, err := solanasdk.FindTimelockSignerPDA(s.TimelockProgramID, testPDASeedTimelockScheduleCancel)
	s.Require().NoError(err)
	e2esolanautils.FundAccounts(s.T(), []solana.PublicKey{mcmSignerPDA, timelockSignerPDA}, 1, s.SolanaClient)

	s.AssignRoleToAccounts(ctx, testPDASeedTimelockScheduleCancel, wallet, []solana.PublicKey{mcmSignerPDA}, timelock.Proposer_Role)
	s.AssignRoleToAccounts(ctx, testPDASeedTimelockScheduleCancel, wallet, []solana.PublicKey{mcmSignerPDA}, timelock.Canceller_Role)

	signer := NewEVMTestAccount(s.T())
	configurer := solanasdk.NewConfigurer(s.SolanaClient, wallet, s.ChainSelector)
	_, err = configurer.SetConfig(ctx, mcmAddress, &types.Config{
		Quorum:  1,
		Signers: []common.Address{signer.Address},
	}, true)
	s.Require().NoError(err)

	inspector := solanasdk.NewInspector(s.SolanaClient)
	opCount, err := inspector.GetOpCount(ctx, mcmAddress)
	s.Require().NoError(err)

	metadata, err := solanasdk.NewChainMetadata(
		opCount,
		s.MCMProgramID,
		testPDASeedTimelockScheduleCancel,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Bypasser_Role].AccessController.PublicKey(),
	)
	s.Require().NoError(err)

	emptyInstruction, err := cpistub.NewEmptyInstruction().ValidateAndBuild()
	s.Require().NoError(err)
	transaction, err := solanasdk.NewTransactionFromInstruction(emptyInstruction, "CPIStub", []string{"cpi-stub-empty"})
	s.Require().NoError(err)

	accessor := chainwrappermocks.NewChainAccessor(s.T())
	accessor.EXPECT().SolanaClient(uint64(s.ChainSelector)).Return(s.SolanaClient, true).Maybe()
	accessor.EXPECT().SolanaSigner(uint64(s.ChainSelector)).Return(&wallet, true).Maybe()

	e2ecommon.RunScheduleAndCancelTest(s.T(), e2ecommon.ScheduleAndCancelTestHooks{
		Setup: func(ctx context.Context, t *testing.T) (e2ecommon.ScheduleAndCancelTestEnv, error) {
			t.Helper()

			return e2ecommon.ScheduleAndCancelTestEnv{
				Proposal: mcms.TimelockProposal{
					BaseProposal: mcms.BaseProposal{
						Version:              "v1",
						Kind:                 types.KindTimelockProposal,
						Description:          "Cancels scheduled Solana timelock CPI stub call",
						ValidUntil:           uint32(time.Now().Add(24 * time.Hour).Unix()),
						OverridePreviousRoot: false,
						Signatures:           []types.Signature{},
						ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
							s.ChainSelector: metadata,
						},
					},
					Action: types.TimelockActionSchedule,
					Delay:  types.MustParseDuration("5m"),
					TimelockAddresses: map[types.ChainSelector]string{
						s.ChainSelector: timelockAddress,
					},
					Operations: []types.BatchOperation{
						{
							ChainSelector: s.ChainSelector,
							Transactions:  []types.Transaction{transaction},
						},
					},
				},
				Chains: accessor,
			}, nil
		},
		Sign: func(t *testing.T, signable *mcms.Signable) {
			t.Helper()
			_, err := signable.SignAndAppend(mcms.NewPrivateKeySigner(signer.PrivateKey))
			require.NoError(t, err)
		},
		WaitForTransaction: func(ctx context.Context, t *testing.T, tx types.TransactionResult) {
			t.Helper()
			_, err := solana.SignatureFromBase58(tx.Hash)
			require.NoError(t, err)
		},
	})
}
