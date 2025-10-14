//go:build e2e

package sui

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

type MCMSUserTestSuite struct {
	SuiTestSuite
}

// SetupSuite runs before the test suite
func (s *MCMSUserTestSuite) SetupSuite() {
	s.SuiTestSuite.SetupSuite()
	s.DeployMCMSContract()
	s.DeployMCMSUserContract()
}

// TestMCMSUserFunctionOne tests MCMS user function one
func (s *MCMSUserTestSuite) Test_MCMSUser_Function_One() {
	s.T().Run("Proposer Role", func(t *testing.T) {
		RunMCMSUserFunctionOneProposal(s, suisdk.TimelockRoleProposer)
	})

	s.T().Run("Bypasser Role", func(t *testing.T) {
		RunMCMSUserFunctionOneProposal(s, suisdk.TimelockRoleBypasser)
	})
}

func RunMCMSUserFunctionOneProposal(s *MCMSUserTestSuite, role suisdk.TimelockRole) {
	bypasserCount := 2
	bypasserQuorum := 2
	bypasserConfig := CreateConfig(bypasserCount, uint8(bypasserQuorum))
	proposerCount := 3
	proposerQuorum := 2
	proposerConfig := CreateConfig(proposerCount, uint8(proposerQuorum))

	// Set config
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, role, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcmsObj, proposerConfig.Config, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.client, s.signer, suisdk.TimelockRoleBypasser, s.mcmsPackageID, s.ownerCapObj, uint64(s.chainSelector))
		s.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.T().Context(), s.mcmsObj, bypasserConfig.Config, true)
		s.Require().NoError(err, "setting config on Sui mcms contract")
	}

	var timelockProposal *mcms.TimelockProposal

	delay5s := time.Second * 5

	// Create a timelock proposal calling MCMS user function one
	// Get the function one call information and build the MCMS Operation
	arg1 := "Updated Field A"
	arg2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	encodedCall, err := s.mcmsUser.Encoder().FunctionOne(
		bind.Object{Id: s.stateObj},
		bind.Object{Id: s.mcmsUserOwnerCapObj},
		arg1,
		arg2,
	)
	s.Require().NoError(err)

	callBytes := s.extractByteArgsFromEncodedCall(*encodedCall)

	transaction, err := suisdk.NewTransactionWithStateObj(
		encodedCall.Module.ModuleName,
		encodedCall.Function,
		encodedCall.Module.PackageID,
		callBytes,
		"MCMSUser",
		[]string{},
		s.stateObj,
	)
	s.Require().NoError(err)

	op := types.BatchOperation{
		ChainSelector: s.chainSelector,
		Transactions:  []types.Transaction{transaction},
	}

	inspector, err := suisdk.NewInspector(s.client, s.signer, s.mcmsPackageID, role)
	s.Require().NoError(err, "creating inspector for op count query")

	// Get the actual current operation count from the contract
	currentOpCount, err := inspector.GetOpCount(s.T().Context(), s.mcmsObj)
	s.Require().NoError(err, "Failed to get current operation count")

	var action types.TimelockAction
	var delay *types.Duration
	switch role {
	case suisdk.TimelockRoleProposer:
		action = types.TimelockActionSchedule
		delayDuration := types.NewDuration(delay5s)
		delay = &delayDuration
	case suisdk.TimelockRoleBypasser:
		action = types.TimelockActionBypass
	case suisdk.TimelockRoleCanceller:
		s.T().Fatalf("TimelockRoleCanceller is not yet supported in this test")
	default:
		s.T().Fatalf("Unsupported role: %v", role)
	}

	proposalConfig := ProposalBuilderConfig{
		Version:        "v1",
		Description:    "MCMS user function one",
		ChainSelector:  s.chainSelector,
		McmsObjID:      s.mcmsObj,
		TimelockObjID:  s.timelockObj,
		McmsPackageID:  s.mcmsPackageID,
		AccountObjID:   s.accountObj,
		RegistryObjID:  s.registryObj,
		Role:           role,
		CurrentOpCount: currentOpCount,
		Action:         action,
		Delay:          delay,
	}

	mcmsUserFunctionOneProposalBuilder := CreateTimelockProposalBuilder(s.T(), proposalConfig, []types.BatchOperation{op})
	timelockProposal, err = mcmsUserFunctionOneProposalBuilder.Build()
	s.Require().NoError(err)

	// Sign the proposal, set root and execute proposal operations

	// Convert the Timelock Proposal into a MCMS Proposal
	timelockConverter, err := suisdk.NewTimelockConverter()
	s.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(s.T().Context(), convertersMap)
	s.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.chainSelector: inspector,
	}

	var keys []*ecdsa.PrivateKey
	var quorum int
	switch role {
	case suisdk.TimelockRoleProposer:
		keys = proposerConfig.Keys
		quorum = proposerQuorum
	case suisdk.TimelockRoleBypasser:
		keys = bypasserConfig.Keys
		quorum = bypasserQuorum
	case suisdk.TimelockRoleCanceller:
		s.T().Fatalf("TimelockRoleCanceller is not yet supported in this test")
	default:
		s.T().Fatalf("Unsupported role: %v", role)
	}
	signable, err := SignProposal(&proposal, inspectorsMap, keys, quorum)
	s.Require().NoError(err)

	// Need to query inspector with MCMS state object ID
	quorumMet, err := signable.ValidateSignatures(s.T().Context())
	s.Require().NoError(err, "Error validating signatures")
	s.Require().True(quorumMet, "Quorum not met")

	// Set Root
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	suiEncoder := encoders[s.chainSelector].(*suisdk.Encoder)
	executor, err := suisdk.NewExecutor(s.client, s.signer, suiEncoder, s.entrypointArgEncoder, s.mcmsPackageID, role, s.mcmsObj, s.accountObj, s.registryObj, s.timelockObj)
	s.Require().NoError(err, "creating executor for Sui mcms contract")

	executors := map[types.ChainSelector]sdk.Executor{
		s.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err, "Error creating executable")

	quorumMet, err = signable.ValidateSignatures(s.T().Context())
	s.Require().NoError(err, "Error validating signatures")
	s.Require().True(quorumMet, "Quorum not met")

	_, err = executable.SetRoot(s.T().Context(), s.chainSelector)
	s.Require().NoError(err)

	for i := range proposal.Operations {
		_, execErr := executable.Execute(s.T().Context(), i)
		s.Require().NoError(execErr)
	}
	if role == suisdk.TimelockRoleProposer {
		// If proposer, some time needs to pass before the proposal can be executed sleep for delay5s
		time.Sleep(delay5s)

		timelockExecutor, tErr := suisdk.NewTimelockExecutor(
			s.client,
			s.signer,
			s.entrypointArgEncoder,
			s.mcmsPackageID,
			s.registryObj,
			s.accountObj,
		)

		s.Require().NoError(tErr, "creating timelock executor for Sui mcms contract")
		timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
			s.chainSelector: timelockExecutor,
		}
		timelockExecutable, execErr := mcms.NewTimelockExecutable(s.T().Context(), timelockProposal, timelockExecutors)
		s.Require().NoError(execErr)

		_, terr := timelockExecutable.Execute(s.T().Context(), 0, mcms.WithCallProxy(s.timelockObj))
		s.Require().NoError(terr)
	}

	// Check op count got incremented by 1
	postOpCount, err := inspector.GetOpCount(s.T().Context(), s.mcmsObj)
	s.Require().NoError(err, "Failed to get post operation count")
	s.Require().Equal(currentOpCount+1, postOpCount, "Operation count should be incremented by 1")

	fieldA, err := s.mcmsUser.DevInspect().GetFieldA(
		s.T().Context(),
		&bind.CallOpts{
			Signer:           s.signer,
			WaitForExecution: true,
		},
		bind.Object{Id: s.stateObj},
	)
	s.Require().NoError(err, "Failed to get fieldA")

	fieldB, err := s.mcmsUser.DevInspect().GetFieldB(
		s.T().Context(),
		&bind.CallOpts{
			Signer:           s.signer,
			WaitForExecution: true,
		},
		bind.Object{Id: s.stateObj},
	)
	s.Require().NoError(err, "Failed to get fieldB")

	s.Require().Equal(arg1, fieldA, "FieldA should be equal to arg1")
	s.Require().Equal(arg2, fieldB, "FieldB should be equal to arg2")
}
