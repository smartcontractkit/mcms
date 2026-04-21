//go:build e2e

package sui

import (
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

func (s *TimelockProposalTestSuite) TestUpdateDelay() {
	ctx := s.T().Context()
	s.DeployMCMSContract()

	timelockInspector, err := suisdk.NewTimelockInspector(s.client, s.signer, s.mcmsPackageID)
	s.Require().NoError(err)

	delay, err := timelockInspector.GetMinDelay(ctx, s.timelockObj)
	s.Require().NoError(err)
	s.Require().EqualValues(0, delay)

	proposerConfig := CreateConfig(1, 1)
	proposerConfigurer, err := suisdk.NewConfigurer(
		s.client,
		s.signer,
		suisdk.TimelockRoleProposer,
		s.mcmsPackageID,
		s.ownerCapObj,
		uint64(s.chainSelector),
	)
	s.Require().NoError(err)

	_, err = proposerConfigurer.SetConfig(ctx, s.mcmsObj, proposerConfig.Config, true)
	s.Require().NoError(err)

	timelockConfigurer := suisdk.NewTimelockConfigurer(s.mcmsPackageID)
	newDelay := uint64(120)

	updateDelayResult, err := timelockConfigurer.UpdateDelay(ctx, s.timelockObj, newDelay)
	s.Require().NoError(err)
	s.Require().Empty(updateDelayResult.Hash)

	updateDelayTx, ok := updateDelayResult.RawData.(types.Transaction)
	s.Require().True(ok, "prepared Sui update delay operation should be an MCMS transaction")

	proposerInspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, suisdk.TimelockRoleProposer)
	s.Require().NoError(err)

	currentOpCount, err := proposerInspector.GetOpCount(ctx, s.mcmsObj)
	s.Require().NoError(err)

	delayDuration := types.NewDuration(2 * time.Second)
	proposalConfig := ProposalBuilderConfig{
		Version:            "v1",
		Description:        "Update timelock minimum delay via timelock configurer",
		ChainSelector:      s.chainSelector,
		McmsObjID:          s.mcmsObj,
		TimelockObjID:      s.timelockObj,
		AccountObjID:       s.accountObj,
		RegistryObjID:      s.registryObj,
		DeployerStateObjID: s.depStateObj,
		McmsPackageID:      s.mcmsPackageID,
		Role:               suisdk.TimelockRoleProposer,
		CurrentOpCount:     currentOpCount,
		Action:             types.TimelockActionSchedule,
		Delay:              &delayDuration,
	}

	timelockProposal, err := CreateTimelockProposalBuilder(s.T(), proposalConfig, []types.BatchOperation{{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{updateDelayTx},
	}}).Build()
	s.Require().NoError(err)

	timelockConverter, err := suisdk.NewTimelockConverter()
	s.Require().NoError(err)

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(ctx, converters)
	s.Require().NoError(err)

	inspectors := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: proposerInspector,
	}
	signable, err := SignProposal(&proposal, inspectors, proposerConfig.Keys, int(proposerConfig.Quorum))
	s.Require().NoError(err)

	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(quorumMet)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	proposalExecutor, err := suisdk.NewExecutor(
		s.client,
		s.signer,
		encoders[s.chainSelector].(*suisdk.Encoder),
		s.entrypointArgEncoder,
		s.mcmsPackageID,
		suisdk.TimelockRoleProposer,
		s.mcmsObj,
		s.accountObj,
		s.registryObj,
		s.timelockObj,
	)
	s.Require().NoError(err)

	executable, err := mcms.NewExecutable(&proposal, map[types.ChainSelector]sdk.Executor{
		s.chainSelector: proposalExecutor,
	})
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.chainSelector)
	s.Require().NoError(err)

	_, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)

	timelockExecutor, err := suisdk.NewTimelockExecutor(
		s.client,
		s.signer,
		s.entrypointArgEncoder,
		s.mcmsPackageID,
		s.registryObj,
		s.accountObj,
	)
	s.Require().NoError(err)

	timelockExecutable, err := mcms.NewTimelockExecutable(ctx, timelockProposal, map[types.ChainSelector]sdk.TimelockExecutor{
		s.chainSelector: timelockExecutor,
	})
	s.Require().NoError(err)

	operationID, err := timelockExecutable.GetOpID(ctx, 0, timelockProposal.Operations[0], s.chainSelector)
	s.Require().NoError(err)

	exists, err := timelockInspector.IsOperation(ctx, s.timelockObj, operationID)
	s.Require().NoError(err)
	s.Require().True(exists)

	s.Require().EventuallyWithT(func(collect *assert.CollectT) {
		assert.NoError(collect, timelockExecutable.IsReady(ctx))
	}, 20*time.Second, time.Second)

	_, err = timelockExecutable.Execute(ctx, 0, mcms.WithCallProxy(s.timelockObj))
	s.Require().NoError(err)

	delay, err = timelockInspector.GetMinDelay(ctx, s.timelockObj)
	s.Require().NoError(err)
	s.Require().Equal(newDelay, delay)
}
