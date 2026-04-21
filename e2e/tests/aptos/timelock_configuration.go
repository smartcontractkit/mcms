//go:build e2e

package aptos

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	aptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

func (a *TestSuite) TestUpdateDelay() {
	ctx := a.T().Context()
	a.deployMCMSContract()

	mcmsAccountAddress := a.MCMSContract.Address()
	mcmsAddress := mcmsAccountAddress.StringLong()
	timelockInspector := aptossdk.NewTimelockInspector(a.AptosRPCClient)

	delay, err := timelockInspector.GetMinDelay(ctx, mcmsAddress)
	a.Require().NoError(err)
	a.Require().EqualValues(0, delay)

	proposerKey, err := crypto.GenerateKey()
	a.Require().NoError(err)
	proposerAddress := crypto.PubkeyToAddress(proposerKey.PublicKey)
	proposerConfig := &types.Config{
		Quorum:  1,
		Signers: []common.Address{proposerAddress},
	}

	proposerConfigurer := aptossdk.NewConfigurer(a.AptosRPCClient, a.deployerAccount, aptossdk.TimelockRoleProposer)
	configResult, err := proposerConfigurer.SetConfig(ctx, mcmsAddress, proposerConfig, false)
	a.Require().NoError(err)

	configTx, err := a.AptosRPCClient.WaitForTransaction(configResult.Hash)
	a.Require().NoError(err)
	a.Require().True(configTx.Success, configTx.VmStatus)

	timelockConfigurer := aptossdk.NewTimelockConfigurer(a.AptosRPCClient)
	newDelay := uint64(120)

	updateDelayResult, err := timelockConfigurer.UpdateDelay(ctx, mcmsAddress, newDelay)
	a.Require().NoError(err)
	a.Require().Empty(updateDelayResult.Hash)

	updateDelayTx, ok := updateDelayResult.RawData.(types.Transaction)
	a.Require().True(ok, "prepared Aptos update delay operation should be an MCMS transaction")

	proposerInspector := aptossdk.NewInspector(a.AptosRPCClient, aptossdk.TimelockRoleProposer)
	startingOpCount, err := proposerInspector.GetOpCount(ctx, mcmsAddress)
	a.Require().NoError(err)

	timelockDelay := types.NewDuration(2 * time.Second)
	timelockProposalBuilder := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(24*time.Hour).Unix())).
		SetDescription("Update timelock minimum delay via timelock configurer").
		AddTimelockAddress(a.ChainSelector, mcmsAddress).
		AddChainMetadata(a.ChainSelector, types.ChainMetadata{
			StartingOpCount:  startingOpCount,
			MCMAddress:       mcmsAddress,
			AdditionalFields: Must(json.Marshal(aptossdk.AdditionalFieldsMetadata{Role: aptossdk.TimelockRoleProposer})),
		}).
		SetAction(types.TimelockActionSchedule).
		SetDelay(timelockDelay).
		AddOperation(types.BatchOperation{
			ChainSelector: a.ChainSelector,
			Transactions:  []types.Transaction{updateDelayTx},
		})

	timelockProposal, err := timelockProposalBuilder.Build()
	a.Require().NoError(err)

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		a.ChainSelector: aptossdk.NewTimelockConverter(),
	}
	proposal, _, err := timelockProposal.Convert(ctx, converters)
	a.Require().NoError(err)

	inspectors := map[types.ChainSelector]sdk.Inspector{
		a.ChainSelector: proposerInspector,
	}
	signable, err := mcms.NewSignable(&proposal, inspectors)
	a.Require().NoError(err)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(proposerKey))
	a.Require().NoError(err)

	quorumMet, err := signable.ValidateSignatures(ctx)
	a.Require().NoError(err)
	a.Require().True(quorumMet)

	encoders, err := proposal.GetEncoders()
	a.Require().NoError(err)

	proposalExecutors := map[types.ChainSelector]sdk.Executor{
		a.ChainSelector: aptossdk.NewExecutor(
			a.AptosRPCClient,
			a.deployerAccount,
			encoders[a.ChainSelector].(*aptossdk.Encoder),
			aptossdk.TimelockRoleProposer,
		),
	}
	executable, err := mcms.NewExecutable(&proposal, proposalExecutors)
	a.Require().NoError(err)

	setRootResult, err := executable.SetRoot(ctx, a.ChainSelector)
	a.Require().NoError(err)

	setRootTx, err := a.AptosRPCClient.WaitForTransaction(setRootResult.Hash)
	a.Require().NoError(err)
	a.Require().True(setRootTx.Success, setRootTx.VmStatus)

	scheduleResult, err := executable.Execute(ctx, 0)
	a.Require().NoError(err)

	scheduleTx, err := a.AptosRPCClient.WaitForTransaction(scheduleResult.Hash)
	a.Require().NoError(err)
	a.Require().True(scheduleTx.Success, scheduleTx.VmStatus)

	timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
		a.ChainSelector: aptossdk.NewTimelockExecutor(a.AptosRPCClient, a.deployerAccount),
	}
	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, timelockProposal, timelockExecutors)
	a.Require().NoError(err)

	operationID, err := timelockExecutable.GetOpID(ctx, 0, timelockProposal.Operations[0], a.ChainSelector)
	a.Require().NoError(err)

	isOperation, err := timelockInspector.IsOperation(ctx, mcmsAddress, operationID)
	a.Require().NoError(err)
	a.Require().True(isOperation)

	a.Require().EventuallyWithT(func(collect *assert.CollectT) {
		assert.NoError(collect, timelockExecutable.IsReady(ctx))
	}, 10*time.Second, 500*time.Millisecond)

	execResult, err := timelockExecutable.Execute(ctx, 0)
	a.Require().NoError(err)

	execTx, err := a.AptosRPCClient.WaitForTransaction(execResult.Hash)
	a.Require().NoError(err)
	a.Require().True(execTx.Success, execTx.VmStatus)

	delay, err = timelockInspector.GetMinDelay(ctx, mcmsAddress)
	a.Require().NoError(err)
	a.Require().Equal(newDelay, delay)
}
