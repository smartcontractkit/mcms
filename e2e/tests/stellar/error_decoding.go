//go:build e2e

package stellare2e

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stellar/go-stellar-sdk/xdr"

	mcmsbindings "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	stellarsdk "github.com/smartcontractkit/mcms/sdk/stellar"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

// TestTimelockExecuteRevertErrorDecoding proves that Stellar timelock
// execution failures preserve and decode the underlying Soroban contract
// error emitted by MCMS.
func (s *ExecutionTestSuite) TestTimelockExecuteRevertErrorDecoding() {
	ctx := s.T().Context()

	// Ownership transfer permanently changes MCMS state, so this test gets
	// an isolated deployment instead of mutating s.Chain.
	chain := s.deployChain()

	s.transferMCMSOwnershipToTimelock(chain)
	s.acceptMCMSOwnershipThroughTimelock(chain)

	inspector := stellarsdk.NewInspectorFromInvoker(
		s.deployer,
	)

	opCount, err := inspector.GetOpCount(
		ctx,
		chain.MCMAddress,
	)
	s.Require().NoError(err)

	revertingTx := s.createRevertingTransaction(
		chain.MCMAddress,
	)

	chainMetadata := map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{s.chainSelector: {StartingOpCount: opCount, MCMAddress: chain.MCMAddress}}

	timelockAddresses := map[mcmtypes.ChainSelector]string{s.chainSelector: chain.TimelockAddress}

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
		common.Hash(tree.Root),
		actualRoot,
		"MCMS root mismatch after SetRoot",
	)
	s.Require().Equal(
		proposal.ValidUntil,
		actualValidUntil,
		"MCMS validUntil mismatch after SetRoot",
	)

	// MCMS executes schedule_batch on timelock. The invalid set_config call
	// is not executed yet, so scheduling must succeed.
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

	timelockExecutable, err := mcms.NewTimelockExecutable(
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

	outer := executionErr.OuterError()
	s.Require().NotNil(outer)
	s.Require().Equal(
		uint32(32),
		outer.Code,
		"outer error should be timelock CallReverted",
	)
	s.Require().Equal(
		"CallReverted",
		outer.Name,
	)
	s.Require().Equal(
		chain.TimelockAddress,
		outer.ContractID,
	)

	root := executionErr.RootCause()
	s.Require().NotNil(root)
	s.Require().Equal(
		uint32(12),
		root.Code,
		"root cause should be MCMS OutOfBoundsGroup",
	)
	s.Require().Equal(
		"OutOfBoundsGroup",
		root.Name,
	)
	s.Require().Equal(
		chain.MCMAddress,
		root.ContractID,
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

	tx, err := stellarsdk.NewTransaction(
		mcmAddress,
		"set_config",
		args,
		"ManyChainMultiSig",
		nil,
	)
	s.Require().NoError(err)

	return tx
}

func (s *ExecutionTestSuite) createScheduleTimelockProposal(
	chainMetadata map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata, timelockAddresses map[mcmtypes.ChainSelector]string,
	batchOps []mcmtypes.BatchOperation,
) mcms.TimelockProposal {
	s.T().Helper()

	return mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmtypes.KindTimelockProposal,
			Description: "Stellar timelock error decoding test",
			ValidUntil: uint32( //nolint:gosec
				time.Now().Add(time.Hour).Unix(),
			),
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata:        chainMetadata,
		},
		Action:            mcmtypes.TimelockActionSchedule,
		Delay:             mcmtypes.MustParseDuration("0s"),
		TimelockAddresses: timelockAddresses,
		Operations:        batchOps,
	}
}
