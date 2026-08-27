package stellar

import (
	"time"

	"github.com/stellar/go-stellar-sdk/xdr"

	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// TestTimelockExecuteRevertErrorDecoding verifies error decoding when a
// scheduled Timelock operation calls MCMS with invalid arguments.
func (s *ExecutionTestSuite) TestTimelockExecuteRevertErrorDecoding() {
	ctx := s.T().Context()

	// The Timelock executes set_config in a separate transaction. Therefore,
	// this path does not reenter an MCMS contract.
	chain := s.deployChain()

	s.transferMCMSOwnershipToTimelock(chain)
	s.acceptMCMSOwnershipThroughTimelock(chain)

	inspector := s.newInspector()

	opCount, err := inspector.GetOpCount(
		ctx,
		chain.MCMAddress,
	)
	s.Require().NoError(err)

	revertingTx := s.createRevertingTransaction(
		chain.MCMAddress,
	)

	chainMetadata := map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
		s.chainSelector: {
			StartingOpCount: opCount,
			MCMAddress:      chain.MCMAddress,
		},
	}

	timelockAddresses := map[mcmtypes.ChainSelector]string{
		s.chainSelector: chain.TimelockAddress,
	}

	batchOps := []mcmtypes.BatchOperation{
		{
			ChainSelector: s.chainSelector,
			Transactions: []mcmtypes.Transaction{
				revertingTx,
			},
		},
	}

	timelockProposal := s.createScheduleTimelockProposal(
		chainMetadata,
		timelockAddresses,
		batchOps,
	)

	proposal, _, err := timelockProposal.Convert(
		ctx,
		map[mcmtypes.ChainSelector]sdk.TimelockConverter{
			s.chainSelector: stellarsdk.NewTimelockConverter(),
		},
	)
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	signable, err := mcms.NewSignable(
		&proposal,
		map[mcmtypes.ChainSelector]sdk.Inspector{
			s.chainSelector: inspector,
		},
	)
	s.Require().NoError(err)

	s.Require().NoError(
		signable.ValidateConfigs(ctx),
	)

	for _, key := range s.proposalSignerKeys {
		_, err = signable.SignAndAppend(
			mcms.NewPrivateKeySigner(key),
		)
		s.Require().NoError(err)
	}

	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(
		quorumMet,
		"MCMS signer quorum was not met",
	)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	encoder, ok := encoders[s.chainSelector].(*stellarsdk.Encoder)
	s.Require().True(
		ok,
		"proposal encoder is not a Stellar encoder",
	)

	executor, err := stellarsdk.NewExecutor(
		encoder,
		inspector,
	)
	s.Require().NoError(err)

	executable, err := mcms.NewExecutable(
		&proposal,
		map[mcmtypes.ChainSelector]sdk.Executor{
			s.chainSelector: executor,
		},
	)
	s.Require().NoError(err)

	_, err = executable.SetRoot(
		ctx,
		s.chainSelector,
	)
	s.Require().NoError(err)

	actualRoot, actualValidUntil, err := inspector.GetRoot(
		ctx,
		chain.MCMAddress,
	)
	s.Require().NoError(err)
	s.Require().Equal(
		tree.Root,
		actualRoot,
		"MCMS root mismatch after SetRoot",
	)
	s.Require().Equal(
		proposal.ValidUntil,
		actualValidUntil,
		"MCMS validUntil mismatch after SetRoot",
	)

	// MCMS calls schedule_batch on Timelock. It does not execute the invalid set_config call yet.
	_, err = executable.Execute(ctx, 0)
	s.Require().NoError(
		err,
		"MCMS should successfully schedule the timelock operation",
	)

	timelockExecutor, err :=
		stellarsdk.NewTimelockExecutorWithNetworkPassphrase(
			s.StellarClient,
			s.StellarSigner,
			s.passphrase,
			s.StellarSigner.Address(),
		)
	s.Require().NoError(err)

	timelockExecutable, err :=
		mcms.NewTimelockExecutable(
			ctx,
			&timelockProposal,
			map[mcmtypes.ChainSelector]sdk.TimelockExecutor{
				s.chainSelector: timelockExecutor,
			},
		)
	s.Require().NoError(err)

	// Timelock now invokes the invalid MCMS set_config call.
	_, err = timelockExecutable.Execute(ctx, 0)
	s.Require().Error(
		err,
		"timelock execution should fail",
	)

	var executionErr *stellarsdk.ExecutionError
	s.Require().ErrorAs(
		err,
		&executionErr,
		"error should contain a Stellar ExecutionError",
	)
	s.Require().NotNil(
		executionErr.Transaction,
		"failed transaction should be preserved",
	)
	s.Require().NotEmpty(
		executionErr.ErrorChain,
		"decoded contract error chain should be preserved",
	)

	outerError := executionErr.OuterError()
	s.Require().NotNil(outerError)
	s.Require().Equal(
		uint32(32),
		outerError.Code,
		"outer error should be Timelock CallReverted",
	)
	s.Require().Equal(
		"CallReverted",
		outerError.Name,
	)
	s.Require().Equal(
		chain.TimelockAddress,
		outerError.ContractID,
	)

	rootCause := executionErr.RootCause()
	s.Require().NotNil(rootCause)
	s.Require().Equal(
		uint32(12),
		rootCause.Code,
		"root cause should be MCMS OutOfBoundsGroup",
	)
	s.Require().Equal(
		"OutOfBoundsGroup",
		rootCause.Name,
	)
	s.Require().Equal(
		chain.MCMAddress,
		rootCause.ContractID,
	)

	s.Require().Contains(
		err.Error(),
		"OutOfBoundsGroup",
	)
}

// TestBypassProposalRevertErrorDecoding verifies error decoding when an MCMS
// operation calls Timelock bypass execution and its nested operation fails.
func (s *ExecutionTestSuite) TestBypassProposalRevertErrorDecoding() {
	ctx := s.T().Context()

	// This MCMS stores the root and executes the converted bypass operation.
	executionChain := s.deployChain()

	// Use another MCMS as the failing target. Calling executionChain.MCMAddress
	// again would cause Soroban contract reentry and return CallAborted.
	targetMCMAddress := s.deployMCMSContract(
		s.nextDeploymentID(),
	)

	targetChain := ChainMeta{
		MCMAddress:      targetMCMAddress,
		TimelockAddress: executionChain.TimelockAddress,
	}

	// Timelock must own the target MCMS. This lets set_config reach its
	// argument validation and return OutOfBoundsGroup.
	s.transferMCMSOwnershipToTimelock(targetChain)
	s.acceptMCMSOwnershipThroughTimelock(targetChain)

	inspector := s.newInspector()

	opCount, err := inspector.GetOpCount(
		ctx,
		executionChain.MCMAddress,
	)
	s.Require().NoError(err)

	revertingTransaction := s.createRevertingTransaction(
		targetMCMAddress,
	)

	timelockProposal := mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmtypes.KindTimelockProposal,
			Description: "Stellar bypass error decoding test",
			ValidUntil: uint32(
				time.Now().Add(time.Hour).Unix(),
			),
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata: map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
				s.chainSelector: {
					StartingOpCount: opCount,
					MCMAddress:      executionChain.MCMAddress,
				},
			},
		},
		Action:            mcmtypes.TimelockActionBypass,
		Delay:             mcmtypes.MustParseDuration("0s"),
		TimelockAddresses: map[mcmtypes.ChainSelector]string{s.chainSelector: executionChain.TimelockAddress},
		Operations: []mcmtypes.BatchOperation{
			{
				ChainSelector: s.chainSelector,
				Transactions: []mcmtypes.Transaction{
					revertingTransaction,
				},
			},
		},
	}

	proposal, _, err := timelockProposal.Convert(
		ctx,
		map[mcmtypes.ChainSelector]sdk.TimelockConverter{
			s.chainSelector: stellarsdk.NewTimelockConverter(),
		},
	)
	s.Require().NoError(err)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	signable, err := mcms.NewSignable(
		&proposal,
		map[mcmtypes.ChainSelector]sdk.Inspector{
			s.chainSelector: inspector,
		},
	)
	s.Require().NoError(err)

	s.Require().NoError(
		signable.ValidateConfigs(ctx),
	)

	for _, key := range s.proposalSignerKeys {
		_, err = signable.SignAndAppend(
			mcms.NewPrivateKeySigner(key),
		)
		s.Require().NoError(err)
	}

	quorumMet, err := signable.ValidateSignatures(ctx)
	s.Require().NoError(err)
	s.Require().True(
		quorumMet,
		"MCMS signer quorum was not met",
	)

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)

	encoder, ok :=
		encoders[s.chainSelector].(*stellarsdk.Encoder)
	s.Require().True(
		ok,
		"proposal encoder is not a Stellar encoder",
	)

	executor, err := stellarsdk.NewExecutor(
		encoder,
		inspector,
	)
	s.Require().NoError(err)

	executable, err := mcms.NewExecutable(
		&proposal,
		map[mcmtypes.ChainSelector]sdk.Executor{
			s.chainSelector: executor,
		},
	)
	s.Require().NoError(err)

	_, err = executable.SetRoot(
		ctx,
		s.chainSelector,
	)
	s.Require().NoError(err)

	actualRoot, actualValidUntil, err := inspector.GetRoot(
		ctx,
		executionChain.MCMAddress,
	)
	s.Require().NoError(err)
	s.Require().Equal(
		tree.Root,
		actualRoot,
		"execution MCMS root mismatch after SetRoot",
	)
	s.Require().Equal(
		proposal.ValidUntil,
		actualValidUntil,
		"execution MCMS validUntil mismatch after SetRoot",
	)

	_, err = executable.Execute(ctx, 0)
	s.Require().Error(
		err,
		"bypass execution should fail",
	)

	var executionErr *stellarsdk.ExecutionError
	s.Require().ErrorAs(
		err,
		&executionErr,
		"error should contain a Stellar ExecutionError",
	)
	s.Require().NotNil(
		executionErr.Transaction,
		"failed transaction should be preserved",
	)
	s.Require().NotEmpty(
		executionErr.ErrorChain,
		"decoded contract error chain should be preserved",
	)

	outerError := executionErr.OuterError()
	s.Require().NotNil(outerError)
	s.Require().Equal(
		uint32(45),
		outerError.Code,
		"outer error should be MCMS CallReverted",
	)
	s.Require().Equal(
		"CallReverted",
		outerError.Name,
	)
	s.Require().Equal(
		executionChain.MCMAddress,
		outerError.ContractID,
	)

	rootCause := executionErr.RootCause()
	s.Require().NotNil(rootCause)
	s.Require().Equal(
		uint32(12),
		rootCause.Code,
		"root cause should be MCMS OutOfBoundsGroup",
	)
	s.Require().Equal(
		"OutOfBoundsGroup",
		rootCause.Name,
	)
	s.Require().Equal(
		targetMCMAddress,
		rootCause.ContractID,
	)

	s.Require().Contains(
		err.Error(),
		"OutOfBoundsGroup",
	)
}

func (s *ExecutionTestSuite) createRevertingTransaction(
	mcmAddress string,
) mcmtypes.Transaction {
	s.T().Helper()

	signerAddresses := mcmsbindings.SignerAddresses{
		Inner: [][32]byte{{1}},
	}
	signerGroups := mcmsbindings.SignerGroups{
		Inner: []uint32{255},
	}

	args := []xdr.ScVal{
		scval.MustToScVal(signerAddresses.ToScVal()),
		scval.MustToScVal(signerGroups.ToScVal()),
		scval.Bytes32ToScVal([32]byte{}),
		scval.Bytes32ToScVal([32]byte{}),
		scval.BoolToScVal(false),
	}

	transaction, err := stellarsdk.NewTransaction(
		mcmAddress,
		"set_config",
		args,
		"ManyChainMultiSig",
		nil,
	)
	s.Require().NoError(err)

	return transaction
}

func (s *ExecutionTestSuite) createScheduleTimelockProposal(
	chainMetadata map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata,
	timelockAddresses map[mcmtypes.ChainSelector]string,
	batchOperations []mcmtypes.BatchOperation,
) mcms.TimelockProposal {
	s.T().Helper()

	return mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmtypes.KindTimelockProposal,
			Description: "Stellar timelock error decoding test",
			ValidUntil: uint32(
				time.Now().Add(time.Hour).Unix(),
			),
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata:        chainMetadata,
		},
		Action:            mcmtypes.TimelockActionSchedule,
		Delay:             mcmtypes.MustParseDuration("0s"),
		TimelockAddresses: timelockAddresses,
		Operations:        batchOperations,
	}
}
