//go:build e2e

package sui

import (
	"crypto/ecdsa"
	"time"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	suisdk "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"

	"github.com/aptos-labs/aptos-go-sdk/bcs"

	"github.com/smartcontractkit/chainlink-sui/bindings/bind"
)

type MCMSUserTestSuite struct {
	SuiTestSuite
}

// SetupSuite runs before the test suite
func (s *MCMSUserTestSuite) SetupSuite() {
	s.SuiTestSuite.SetupSuite()
	s.SuiTestSuite.DeployMCMSContract()
	s.SuiTestSuite.DeployMCMSUserContract()
}

// TestMCMSUserFunctionOne tests MCMS user function one
func (s *MCMSUserTestSuite) Test_MCMSUser_Function_One() {
	RunMCMSUserFunctionOneProposal(s, suisdk.TimelockRoleProposer)
	RunMCMSUserFunctionOneProposal(s, suisdk.TimelockRoleBypasser)
}

func RunMCMSUserFunctionOneProposal(s *MCMSUserTestSuite, role suisdk.TimelockRole) {
	s.SuiTestSuite.T().Logf("Running MCMS user function one proposal with role: %v", role)

	proposerConfig := CreateProposerConfig(3, 2)
	bypasserConfig := CreateBypasserConfig(2, 2)

	// Set config
	{
		configurer, err := suisdk.NewConfigurer(s.SuiTestSuite.client, s.SuiTestSuite.signer, role, s.SuiTestSuite.mcmsPackageId, s.SuiTestSuite.ownerCapObj, uint64(s.SuiTestSuite.chainSelector))
		s.SuiTestSuite.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj, proposerConfig.Config, true)
		s.SuiTestSuite.Require().NoError(err, "setting config on Sui mcms contract")
	}
	{
		configurer, err := suisdk.NewConfigurer(s.SuiTestSuite.client, s.SuiTestSuite.signer, suisdk.TimelockRoleBypasser, s.SuiTestSuite.mcmsPackageId, s.SuiTestSuite.ownerCapObj, uint64(s.SuiTestSuite.chainSelector))
		s.SuiTestSuite.Require().NoError(err, "creating configurer for Sui mcms contract")
		_, err = configurer.SetConfig(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj, bypasserConfig.Config, true)
		s.SuiTestSuite.Require().NoError(err, "setting config on Sui mcms contract")
	}

	var timelockProposal *mcms.TimelockProposal

	delay_5_secs := time.Second * 5

	// Create a timelock proposal calling MCMS user function one
	// Get the function one call information and build the MCMS Operation
	arg1 := "Updated Field A"
	arg2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	encodedCall, err := s.SuiTestSuite.mcmsUser.Encoder().FunctionOne(
		bind.Object{Id: s.SuiTestSuite.stateObj},
		bind.Object{Id: s.SuiTestSuite.mcmsUserOwnerCapObj},
		arg1,
		arg2,
	)
	s.SuiTestSuite.Require().NoError(err)

	// TODO: We should construct the mcms tx data using the bindings
	callBytes, err := s.serializeFunctionOneData(arg1, arg2)
	s.SuiTestSuite.Require().NoError(err, "Failed to serialize function one data")

	transaction, err := suisdk.NewTransactionWithStateObj(
		encodedCall.Module.ModuleName,
		encodedCall.Function,
		encodedCall.Module.PackageID,
		callBytes,
		"MCMSUser",
		[]string{},
		s.SuiTestSuite.stateObj,
	)
	s.SuiTestSuite.Require().NoError(err)

	op := types.BatchOperation{
		ChainSelector: s.SuiTestSuite.chainSelector,
		Transactions:  []types.Transaction{transaction},
	}

	inspector, err := suisdk.NewInspector(s.SuiTestSuite.client, s.SuiTestSuite.signer, s.SuiTestSuite.mcmsPackageId, role)
	s.SuiTestSuite.Require().NoError(err, "creating inspector for op count query")

	// Get the actual current operation count from the contract
	currentOpCount, err := inspector.GetOpCount(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj)
	s.SuiTestSuite.Require().NoError(err, "Failed to get current operation count")
	s.SuiTestSuite.T().Logf("🔍 CURRENT operation count: %d", currentOpCount)

	var action types.TimelockAction
	var delay *types.Duration
	if role == suisdk.TimelockRoleProposer {
		action = types.TimelockActionSchedule
		delayDuration := types.NewDuration(delay_5_secs)
		delay = &delayDuration
	} else if role == suisdk.TimelockRoleBypasser {
		action = types.TimelockActionBypass
	} else {
		s.SuiTestSuite.T().Fatalf("Unsupported role: %v", role)
	}

	proposalConfig := ProposalBuilderConfig{
		Version:        "v1",
		Description:    "MCMS user function one",
		ChainSelector:  s.SuiTestSuite.chainSelector,
		MCMSPackageId:  s.SuiTestSuite.mcmsPackageId,
		Role:           role,
		CurrentOpCount: currentOpCount,
		Action:         action,
		Delay:          delay,
	}

	mcmsUserFunctionOneProposalBuilder := CreateTimelockProposalBuilder(proposalConfig, []types.BatchOperation{op})
	timelockProposal, err = mcmsUserFunctionOneProposalBuilder.Build()
	s.SuiTestSuite.Require().NoError(err)

	// Sign the proposal, set root and execute proposal operations

	// Convert the Timelock Proposal into a MCMS Proposal
	timelockConverter, err := suisdk.NewTimelockConverter(s.SuiTestSuite.client, s.SuiTestSuite.signer, s.SuiTestSuite.mcmsPackageId)
	s.SuiTestSuite.Require().NoError(err)

	convertersMap := map[types.ChainSelector]sdk.TimelockConverter{
		s.SuiTestSuite.chainSelector: timelockConverter,
	}
	proposal, _, err := timelockProposal.Convert(s.SuiTestSuite.T().Context(), convertersMap)
	s.SuiTestSuite.Require().NoError(err)

	inspectorsMap := map[types.ChainSelector]sdk.Inspector{
		s.SuiTestSuite.chainSelector: inspector,
	}

	s.SuiTestSuite.T().Logf("Signing the proposal...")

	var keys []*ecdsa.PrivateKey
	var quorum int
	if role == suisdk.TimelockRoleProposer {
		keys = proposerConfig.Keys
		quorum = proposerQuorum
	} else if role == suisdk.TimelockRoleBypasser {
		keys = bypasserConfig.Keys
	} else {
		s.SuiTestSuite.T().Fatalf("Unsupported role: %v", role)
	}
	signable, err := SignProposal(&proposal, inspectorsMap, keys, quorum)
	s.SuiTestSuite.Require().NoError(err)

	// Need to query inspector with MCMS state object ID
	quorumMet, err := signable.ValidateSignaturesWithMCMAddress(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj)
	s.SuiTestSuite.Require().NoError(err, "Error validating signatures")
	s.SuiTestSuite.Require().True(quorumMet, "Quorum not met")

	// Set Root
	s.SuiTestSuite.T().Logf("Preparing to the root of the proposal...")
	encoders, err := proposal.GetEncoders()
	s.SuiTestSuite.Require().NoError(err)
	suiEncoder := encoders[s.SuiTestSuite.chainSelector].(*suisdk.Encoder)
	executor, err := suisdk.NewExecutor(s.SuiTestSuite.client, s.SuiTestSuite.signer, suiEncoder, s.SuiTestSuite.mcmsPackageId, role, s.SuiTestSuite.mcmsObj, s.SuiTestSuite.accountObj, s.SuiTestSuite.registryObj, s.SuiTestSuite.timelockObj)
	s.SuiTestSuite.Require().NoError(err, "creating executor for Sui mcms contract")

	executors := map[types.ChainSelector]sdk.Executor{
		s.SuiTestSuite.chainSelector: executor,
	}
	executable, err := mcms.NewExecutable(&proposal, executors)
	s.SuiTestSuite.Require().NoError(err, "Error creating executable")

	s.SuiTestSuite.T().Logf("Setting the root of the proposal...")

	s.SuiTestSuite.T().Logf("=== DEBUG: Proposal Details ===")
	merkleTree, err := proposal.MerkleTree()
	s.SuiTestSuite.Require().NoError(err, "Failed to get merkle tree")
	s.SuiTestSuite.T().Logf("Merkle Tree Root: %x", merkleTree.Root)
	s.SuiTestSuite.T().Logf("Proposal ValidUntil: %d", proposal.ValidUntil)
	s.SuiTestSuite.T().Logf("Number of Operations: %d", len(proposal.Operations))

	for chainSel, metadata := range proposal.ChainMetadata {
		s.SuiTestSuite.T().Logf("Chain %d metadata - StartingOpCount: %d, MCMAddress: %s",
			chainSel, metadata.StartingOpCount, metadata.MCMAddress)
	}

	for i, op := range proposal.Operations {
		s.SuiTestSuite.T().Logf("Operation %d: ChainSelector=%d, To=%s, DataLen=%d",
			i, op.ChainSelector, op.Transaction.To, len(op.Transaction.Data))
	}

	signingHash, err := proposal.SigningHash()
	s.SuiTestSuite.Require().NoError(err, "Failed to get signing hash")
	s.SuiTestSuite.T().Logf("Proposal Signing Hash: %x", signingHash)

	quorumMet, err = signable.ValidateSignaturesWithMCMAddress(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj)
	s.SuiTestSuite.Require().NoError(err, "Error validating signatures")
	s.SuiTestSuite.Require().True(quorumMet, "Quorum not met")

	result, err := executable.SetRoot(s.SuiTestSuite.T().Context(), s.SuiTestSuite.chainSelector)
	s.SuiTestSuite.Require().NoError(err)

	s.SuiTestSuite.T().Logf("✅ SetRoot in tx: %s", result.Hash)

	s.SuiTestSuite.T().Logf("Executing the proposal operations...")

	for i := range proposal.Operations {
		s.SuiTestSuite.T().Logf("Executing operation: %v", i)
		txOutput, err := executable.Execute(s.SuiTestSuite.T().Context(), i)
		s.SuiTestSuite.Require().NoError(err)
		s.SuiTestSuite.T().Logf("✅ Executed Operation in tx: %s", txOutput.Hash)
	}
	if role == suisdk.TimelockRoleProposer {
		// If proposer, some time needs to pass before the proposal can be executed sleep for delay_5_secs
		s.SuiTestSuite.T().Logf("Sleeping for %v before executing the proposal transfer...", delay_5_secs)
		time.Sleep(delay_5_secs)

		timelockExecutor, err := suisdk.NewTimelockExecutor(
			s.SuiTestSuite.client,
			s.SuiTestSuite.signer,
			s.SuiTestSuite.mcmsPackageId,
			s.SuiTestSuite.registryObj,
			s.SuiTestSuite.accountObj,
		)

		s.SuiTestSuite.Require().NoError(err, "creating timelock executor for Sui mcms contract")
		timelockExecutors := map[types.ChainSelector]sdk.TimelockExecutor{
			s.SuiTestSuite.chainSelector: timelockExecutor,
		}
		timelockExecutable, err := mcms.NewTimelockExecutable(s.SuiTestSuite.T().Context(), timelockProposal, timelockExecutors)
		s.SuiTestSuite.Require().NoError(err)

		s.SuiTestSuite.T().Logf("Executing the operation through timelock...")
		txOutput, err := timelockExecutable.Execute(s.SuiTestSuite.T().Context(), 0, mcms.WithCallProxy(s.SuiTestSuite.timelockObj))
		s.SuiTestSuite.Require().NoError(err)
		s.SuiTestSuite.T().Logf("✅ Executed proposal transfer in tx: %s", txOutput.Hash)
	}

	// Check op count got incremented by 1
	postOpCount, err := inspector.GetOpCount(s.SuiTestSuite.T().Context(), s.SuiTestSuite.mcmsObj)
	s.SuiTestSuite.Require().NoError(err, "Failed to get post operation count")
	s.SuiTestSuite.Require().Equal(currentOpCount+1, postOpCount, "Operation count should be incremented by 1")

	fieldA, err := s.SuiTestSuite.mcmsUser.DevInspect().GetFieldA(
		s.SuiTestSuite.T().Context(),
		&bind.CallOpts{
			Signer:           s.SuiTestSuite.signer,
			WaitForExecution: true,
		},
		bind.Object{Id: s.SuiTestSuite.stateObj},
	)
	s.SuiTestSuite.Require().NoError(err, "Failed to get fieldA")

	fieldB, err := s.SuiTestSuite.mcmsUser.DevInspect().GetFieldB(
		s.SuiTestSuite.T().Context(),
		&bind.CallOpts{
			Signer:           s.SuiTestSuite.signer,
			WaitForExecution: true,
		},
		bind.Object{Id: s.SuiTestSuite.stateObj},
	)
	s.SuiTestSuite.Require().NoError(err, "Failed to get fieldB")

	s.SuiTestSuite.T().Logf("🔍 New fieldA: %s", fieldA)
	s.SuiTestSuite.T().Logf("🔍 New fieldB: %x", fieldB)
	s.SuiTestSuite.Require().Equal(arg1, fieldA, "FieldA should be equal to arg1")
	s.SuiTestSuite.Require().Equal(arg2, fieldB, "FieldB should be equal to arg2")
	s.SuiTestSuite.T().Logf("✅ Successfully executed MCMS user function one proposal with role: %v", role)
}

func (s *MCMSUserTestSuite) serializeFunctionOneData(arg1 string, arg2 []byte) ([]byte, error) {
	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.WriteString(arg1)
		ser.WriteBytes(arg2)
	})
}
