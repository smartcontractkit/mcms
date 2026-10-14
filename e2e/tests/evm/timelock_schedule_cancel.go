//go:build e2e

package evme2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	chainwrappermocks "github.com/smartcontractkit/mcms/chainwrappers/mocks"
	e2ecommon "github.com/smartcontractkit/mcms/e2e/tests/common"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

func (s *ExecutionTestSuite) TestScheduleAndCancelProposal() {
	ctx := s.T().Context()

	target := s.signerAddresses[0]
	role, err := s.ChainA.timelockContract.EXECUTORROLE(&bind.CallOpts{Context: ctx})
	s.Require().NoError(err)

	hasRole, err := s.ChainA.timelockContract.HasRole(&bind.CallOpts{Context: ctx}, role, target)
	s.Require().NoError(err)
	s.Require().False(hasRole, "target should not already have executor role")

	inspector := evm.NewInspector(s.ClientA)
	opCount, err := inspector.GetOpCount(ctx, s.ChainA.mcmsContract.Address().Hex())
	s.Require().NoError(err)

	timelockABI, err := bindings.RBACTimelockMetaData.GetAbi()
	s.Require().NoError(err)
	grantRoleData, err := timelockABI.Pack("grantRole", role, target)
	s.Require().NoError(err)

	accessor := chainwrappermocks.NewChainAccessor(s.T())
	accessor.EXPECT().EVMClient(uint64(s.ChainA.chainSelector)).Return(s.ClientA, true).Maybe()
	accessor.EXPECT().EVMSigner(uint64(s.ChainA.chainSelector)).Return(s.ChainA.auth, true).Maybe()

	signer := testutils.ParsePrivateKey(s.Settings.PrivateKeys[1])
	e2ecommon.RunScheduleAndCancelTest(s.T(), e2ecommon.ScheduleAndCancelTestHooks{
		Setup: func(ctx context.Context, t *testing.T) (e2ecommon.ScheduleAndCancelTestEnv, error) {
			t.Helper()

			return e2ecommon.ScheduleAndCancelTestEnv{
				Proposal: mcms.TimelockProposal{
					BaseProposal: mcms.BaseProposal{
						Version:              "v1",
						Kind:                 mcmtypes.KindTimelockProposal,
						Description:          "Cancels scheduled EVM timelock role grant",
						ValidUntil:           uint32(time.Now().Add(24 * time.Hour).Unix()),
						OverridePreviousRoot: false,
						Signatures:           []mcmtypes.Signature{},
						ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
							s.ChainA.chainSelector: {
								StartingOpCount: opCount,
								MCMAddress:      s.ChainA.mcmsContract.Address().Hex(),
							},
						},
					},
					Action: mcmtypes.TimelockActionSchedule,
					Delay:  mcmtypes.MustParseDuration("5m"),
					TimelockAddresses: map[mcmtypes.ChainSelector]string{
						s.ChainA.chainSelector: s.ChainA.timelockContract.Address().Hex(),
					},
					Operations: []mcmtypes.BatchOperation{
						{
							ChainSelector: s.ChainA.chainSelector,
							Transactions: []mcmtypes.Transaction{
								evm.NewTransaction(
									common.HexToAddress(s.ChainA.timelockContract.Address().Hex()),
									grantRoleData,
									big.NewInt(0),
									"RBACTimelock",
									[]string{"RBACTimelock", "GrantRole"},
								),
							},
						},
					},
				},
				Chains: accessor,
			}, nil
		},
		Sign: func(t *testing.T, signable *mcms.Signable) {
			t.Helper()
			_, err := signable.SignAndAppend(mcms.NewPrivateKeySigner(signer))
			require.NoError(t, err)
		},
		WaitForTransaction: func(ctx context.Context, t *testing.T, tx mcmtypes.TransactionResult) {
			t.Helper()
			receipt, err := testutils.WaitMinedWithTxHash(ctx, s.ClientA, common.HexToHash(tx.Hash))
			require.NoError(t, err)
			require.Equal(t, gethtypes.ReceiptStatusSuccessful, receipt.Status)
		},
		AssertExtraAfterCancel: func(ctx context.Context, t *testing.T, env *e2ecommon.ScheduleAndCancelTestEnv) {
			t.Helper()
			hasRole, err := s.ChainA.timelockContract.HasRole(&bind.CallOpts{Context: ctx}, role, target)
			require.NoError(t, err)
			require.False(t, hasRole, "target should not receive role after cancellation")
		},
	})
}
