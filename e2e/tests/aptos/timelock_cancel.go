//go:build e2e

package aptos

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"

	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"

	mcmslib "github.com/smartcontractkit/mcms"
	chainwrappermocks "github.com/smartcontractkit/mcms/chainwrappers/mocks"
	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/types"
)

func (a *TestSuite) TestTimelock_Cancel() {
	a.deployMCMSContract()
	mcmsAddress := a.MCMSContract.Address()
	a.initTransferOwnership()

	// Generate signers for both Proposer and Canceller roles
	proposerSigners := testutils.MakeNewECDSASigners(2)
	cancellerSigners := testutils.MakeNewECDSASigners(2)

	a.configureRole(aptossdk.TimelockRoleProposer, keysToAddresses(proposerSigners), 2)
	a.configureRole(aptossdk.TimelockRoleCanceller, keysToAddresses(cancellerSigners), 2)

	accessor := chainwrappermocks.NewChainAccessor(a.T())
	accessor.EXPECT().AptosClient(uint64(a.ChainSelector)).Return(a.AptosRPCClient, true).Maybe()
	accessor.EXPECT().AptosSigner(uint64(a.ChainSelector)).Return(a.deployerAccount, true).Maybe()

	phase := 0
	mcmslib.RunScheduleAndCancelTest(a.T(), mcmslib.ScheduleAndCancelTestHooks{
		Setup: func(ctx context.Context, t *testing.T) (mcmslib.ScheduleAndCancelTestEnv, error) {
			transaction, txErr := a.buildAcceptOwnershipTransaction()
			if txErr != nil {
				return mcmslib.ScheduleAndCancelTestEnv{}, txErr
			}

			return mcmslib.ScheduleAndCancelTestEnv{
				Proposal: mcmslib.TimelockProposal{
					BaseProposal: mcmslib.BaseProposal{
						Version:     "v1",
						Kind:        types.KindTimelockProposal,
						Description: "Accept ownership via timelock",
						ValidUntil:  uint32(time.Now().Add(time.Hour * 24).Unix()),
						ChainMetadata: map[types.ChainSelector]types.ChainMetadata{
							a.ChainSelector: {
								StartingOpCount:  0,
								MCMAddress:       mcmsAddress.StringLong(),
								AdditionalFields: Must(json.Marshal(aptossdk.AdditionalFieldsMetadata{Role: aptossdk.TimelockRoleProposer})),
							},
						},
					},
					Action: types.TimelockActionSchedule,
					Delay:  types.MustParseDuration("1s"),
					TimelockAddresses: map[types.ChainSelector]string{
						a.ChainSelector: mcmsAddress.StringLong(),
					},
					Operations: []types.BatchOperation{
						{
							ChainSelector: a.ChainSelector,
							Transactions:  []types.Transaction{transaction},
						},
					},
				},
				Chains: accessor,
			}, nil
		},
		Sign: func(t *testing.T, signable *mcmslib.Signable) {
			// Phase 0 = schedule (proposer), Phase 1 = cancel (canceller)
			signers := proposerSigners
			if phase > 0 {
				signers = cancellerSigners
			}
			phase++
			for _, signer := range signers {
				_, err := signable.SignAndAppend(mcmslib.NewPrivateKeySigner(signer.Key))
				require.NoError(t, err)
			}
		},
		DeriveCancellationMetadata: func(t *testing.T, selector types.ChainSelector, scheduleMetadata types.ChainMetadata) (types.ChainMetadata, error) {
			// For the cancellation proposal, the MCMS role in AdditionalFields must be switched
			// from the schedule role (Proposer) to the Canceller role. The Aptos SetRoot call
			// reads the role from this metadata field and passes it to the on-chain contract.
			next := scheduleMetadata
			next.AdditionalFields = Must(json.Marshal(aptossdk.AdditionalFieldsMetadata{Role: aptossdk.TimelockRoleCanceller}))

			cancellerInspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleCanceller)
			cancellerOpCount, err := cancellerInspector.GetOpCount(t.Context(), next.MCMAddress)
			if err != nil {
				return types.ChainMetadata{}, err
			}
			next.StartingOpCount = cancellerOpCount

			return next, nil
		},
		WaitForTransaction: func(ctx context.Context, t *testing.T, tx types.TransactionResult) {
			data, err := a.AptosRPCClient.WaitForTransaction(tx.Hash)
			require.NoError(t, err)
			require.True(t, data.Success, data.VmStatus)
		},
	})
}

func (a *TestSuite) initTransferOwnership() {
	opts := &bind.TransactOpts{Signer: a.deployerAccount}
	tx, err := a.MCMSContract.MCMSAccount().TransferOwnershipToSelf(opts)
	a.Require().NoError(err)
	data, err := a.AptosRPCClient.WaitForTransaction(tx.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
}

func (a *TestSuite) configureRole(role aptossdk.TimelockRole, signers []common.Address, quorum uint8) {
	config := &types.Config{
		Quorum:  quorum,
		Signers: signers,
	}
	configurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, role)
	addr := a.MCMSContract.Address()
	result, err := configurer.SetConfig(a.T().Context(), addr.StringLong(), config, false)
	a.Require().NoError(err)
	data, err := a.AptosRPCClient.WaitForTransaction(result.Hash)
	a.Require().NoError(err)
	a.Require().True(data.Success, data.VmStatus)
}

func (a *TestSuite) buildAcceptOwnershipTransaction() (types.Transaction, error) {
	module, function, _, args, err := a.MCMSContract.MCMSAccount().Encoder().AcceptOwnership()
	if err != nil {
		return types.Transaction{}, err
	}
	return aptossdk.NewTransaction(
		module.PackageName,
		module.ModuleName,
		function,
		a.MCMSContract.Address(),
		aptossdk.ArgsToData(args),
		"MCMS",
		nil,
	)
}

func keysToAddresses(signers []testutils.ECDSASigner) []common.Address {
	addrs := make([]common.Address, len(signers))
	for i, s := range signers {
		addrs[i] = s.Address()
	}
	return addrs
}
