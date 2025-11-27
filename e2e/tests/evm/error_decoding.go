//go:build e2e
// +build e2e

package evme2e

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	testutils "github.com/smartcontractkit/mcms/e2e/utils"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	mcmtypes "github.com/smartcontractkit/mcms/types"
)

const mcmsABIErrorMsg = "Failed to get MCMS ABI"

// TestTimelockExecuteRevertErrorDecoding tests that error decoding works correctly
// for timelock execute calls
func (s *ExecutionTestSuite) TestTimelockExecuteRevertErrorDecoding() {
	ctx := s.T().Context()

	mcmsContract := s.deployMCMSContract(s.ChainA.auth, s.ClientA)
	roleCfg := s.defaultTimelockRoleConfig(mcmsContract.Address(), s.ChainA.auth.From)
	timelockContract := s.deployTimelockContract(s.ChainA.auth, s.ClientA, mcmsContract.Address().Hex(), roleCfg)

	transferMCMSOwnershipToTimelock(
		s.T(),
		ctx,
		mcmsContract,
		timelockContract.Address(),
		s.ChainA.auth,
		s.ClientA,
	)

	acceptMCMSOwnership(
		s.T(),
		ctx,
		mcmsContract,
		timelockContract,
		s.ChainA.auth,
		s.ClientA,
	)

	opts := &bind.CallOpts{
		Context:     ctx,
		From:        s.ChainA.auth.From,
		BlockNumber: nil,
	}
	opCount, err := mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)

	// Create a transaction that calls SetConfig on MCMS with invalid data
	// This will revert when executed by the timelock, allowing us to test error decoding
	revertingTx := createRevertingTransaction(s.T(), mcmsContract.Address())

	chainMetadata := map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
		s.ChainA.chainSelector: {
			StartingOpCount: opCount.Uint64(),
			MCMAddress:      mcmsContract.Address().Hex(),
		},
	}
	timelockAddresses := map[mcmtypes.ChainSelector]string{
		s.ChainA.chainSelector: timelockContract.Address().Hex(),
	}
	batchOps := []mcmtypes.BatchOperation{
		{
			ChainSelector: s.ChainA.chainSelector,
			Transactions:  []mcmtypes.Transaction{revertingTx},
		},
	}

	timelockProposal := createScheduleTimelockProposal(s.T(), chainMetadata, timelockAddresses, batchOps)

	converters := map[mcmtypes.ChainSelector]sdk.TimelockConverter{
		s.ChainA.chainSelector: &evm.TimelockConverter{},
	}
	proposal, _ := convertTimelockProposal(s.T(), ctx, timelockProposal, converters)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.ChainA.chainSelector: evm.NewInspector(s.ClientA),
	}

	_ = signAndValidateProposal(s.T(), ctx, &proposal, inspectors, []string{
		s.Settings.PrivateKeys[1],
		s.Settings.PrivateKeys[2],
	})

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		s.ChainA.chainSelector: evm.NewExecutor(
			encoders[s.ChainA.chainSelector].(*evm.Encoder),
			s.ClientA,
			s.ChainA.auth,
		),
	}

	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	setRootAndVerify(
		s.T(),
		ctx,
		executable,
		s.ChainA.chainSelector,
		[32]byte(tree.Root),
		proposal.ValidUntil,
		s.ClientA,
		mcmsContract,
	)

	_, err = executable.Execute(ctx, 0)
	s.T().Logf("[TestTimelockExecute] MCMS Execute returned error: %v", err)
	s.Require().NoError(err)

	timelockExecutors := map[mcmtypes.ChainSelector]sdk.TimelockExecutor{
		s.ChainA.chainSelector: evm.NewTimelockExecutor(
			s.ClientA,
			s.ChainA.auth,
		),
	}

	timelockExecutable, err := mcms.NewTimelockExecutable(s.T().Context(), &timelockProposal, timelockExecutors)
	s.Require().NoError(err)

	_, err = timelockExecutable.Execute(s.T().Context(), 0)
	s.T().Logf("[TestTimelockExecute] Timelock Execute returned error: %v", err)
	s.Require().Error(err)

	mcmsABI, abiErr := bindings.ManyChainMultiSigMetaData.GetAbi()
	s.Require().NoError(abiErr, mcmsABIErrorMsg)
	outOfBoundsGroupError, exists := mcmsABI.Errors["OutOfBoundsGroup"]
	s.Require().True(exists, "OutOfBoundsGroup error not found in MCMS ABI")
	var expectedSelector [4]byte
	copy(expectedSelector[:], outOfBoundsGroupError.ID[:4])
	timelockExecErr := assertExecutionError(
		s.T(),
		err,
		true,
		true,
		"OutOfBoundsGroup",
		[4]byte{},
	)

	s.Require().NotNil(timelockExecErr, "ExecutionError should not be nil")
	s.Require().NotNil(timelockExecErr.Transaction, "Transaction should be set in ExecutionError")
	s.Require().NotEmpty(timelockExecErr.UnderlyingReasonRaw, "UnderlyingReasonRaw should be extracted")
	s.Require().NotEmpty(timelockExecErr.UnderlyingReasonDecoded, "UnderlyingReasonDecoded should be extracted")

	s.Contains(err.Error(), "OutOfBoundsGroup")
	s.Equal(outOfBoundsGroupError.Name, timelockExecErr.UnderlyingReasonDecoded, "Decoded underlying reason should match MCMS error")
	s.T().Logf("[TestTimelockExecute] ExecutionError details: Transaction=%v RawRevert=%v Decoded=%q UnderlyingRaw=%q UnderlyingDecoded=%q",
		timelockExecErr.Transaction,
		timelockExecErr.RevertReasonRaw,
		timelockExecErr.RevertReasonDecoded,
		timelockExecErr.UnderlyingReasonRaw,
		timelockExecErr.UnderlyingReasonDecoded,
	)
}

// TestBypassProposalRevertErrorDecoding tests that error decoding works correctly
// when a bypass proposal's underlying transaction reverts.
func (s *ExecutionTestSuite) TestBypassProposalRevertErrorDecoding() {
	ctx := s.T().Context()

	mcmsContract := s.deployMCMSContract(s.ChainA.auth, s.ClientA)
	roleCfg := s.defaultTimelockRoleConfig(mcmsContract.Address(), s.ChainA.auth.From)
	timelockContract := s.deployTimelockContract(s.ChainA.auth, s.ClientA, mcmsContract.Address().Hex(), roleCfg)

	transferMCMSOwnershipToTimelock(
		s.T(),
		ctx,
		mcmsContract,
		timelockContract.Address(),
		s.ChainA.auth,
		s.ClientA,
	)

	acceptMCMSOwnership(
		s.T(),
		ctx,
		mcmsContract,
		timelockContract,
		s.ChainA.auth,
		s.ClientA,
	)

	opts := &bind.CallOpts{
		Context:     ctx,
		From:        s.ChainA.auth.From,
		BlockNumber: nil,
	}
	opCount, err := mcmsContract.GetOpCount(opts)
	s.Require().NoError(err)

	revertingTx := createRevertingTransaction(s.T(), mcmsContract.Address())

	chainMetadata := map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata{
		s.ChainA.chainSelector: {
			StartingOpCount: opCount.Uint64(),
			MCMAddress:      mcmsContract.Address().Hex(),
		},
	}
	timelockAddresses := map[mcmtypes.ChainSelector]string{
		s.ChainA.chainSelector: timelockContract.Address().Hex(),
	}
	batchOps := []mcmtypes.BatchOperation{
		{
			ChainSelector: s.ChainA.chainSelector,
			Transactions:  []mcmtypes.Transaction{revertingTx},
		},
	}

	timelockProposal := createBypassTimelockProposal(s.T(), chainMetadata, timelockAddresses, batchOps)

	converters := map[mcmtypes.ChainSelector]sdk.TimelockConverter{
		s.ChainA.chainSelector: &evm.TimelockConverter{},
	}
	proposal, _ := convertTimelockProposal(s.T(), ctx, timelockProposal, converters)

	tree, err := proposal.MerkleTree()
	s.Require().NoError(err)

	inspectors := map[mcmtypes.ChainSelector]sdk.Inspector{
		s.ChainA.chainSelector: evm.NewInspector(s.ClientA),
	}
	_ = signAndValidateProposal(s.T(), ctx, &proposal, inspectors, []string{
		s.Settings.PrivateKeys[1], // Signer for Group 0
		s.Settings.PrivateKeys[2], // Signer for Group 1
	})

	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	executors := map[mcmtypes.ChainSelector]sdk.Executor{
		s.ChainA.chainSelector: evm.NewExecutor(
			encoders[s.ChainA.chainSelector].(*evm.Encoder),
			s.ClientA,
			s.ChainA.auth,
		),
	}

	executable, err := mcms.NewExecutable(&proposal, executors)
	s.Require().NoError(err)

	setRootAndVerify(
		s.T(),
		ctx,
		executable,
		s.ChainA.chainSelector,
		[32]byte(tree.Root),
		proposal.ValidUntil,
		s.ClientA,
		mcmsContract,
	)

	_, err = executable.Execute(ctx, 0)
	s.Require().Error(err, "Execute should fail with revert")

	mcmsABI, errABI := bindings.ManyChainMultiSigMetaData.GetAbi()
	s.Require().NoError(errABI, mcmsABIErrorMsg)
	outOfBoundsGroupError, exists := mcmsABI.Errors["OutOfBoundsGroup"]
	s.Require().True(exists, "OutOfBoundsGroup error not found in MCMS ABI")

	execErr := assertExecutionError(
		s.T(),
		err,
		true,
		true,
		"OutOfBoundsGroup",
		evm.CallRevertedSelector,
	)
	s.Require().NotNil(execErr, "ExecutionError should not be nil")
	s.Require().NotNil(execErr.Transaction, "Transaction should be set in ExecutionError")
	s.Require().NotEmpty(execErr.UnderlyingReasonRaw, "UnderlyingReasonRaw should be extracted")
	s.Require().NotEmpty(execErr.UnderlyingReasonDecoded, "UnderlyingReasonDecoded should be extracted")

	s.Contains(err.Error(), "OutOfBoundsGroup")
	s.NotEmpty(execErr.UnderlyingReasonRaw, "UnderlyingReasonRaw should not be empty")
	s.NotNil(execErr.RevertReasonRaw, "RevertReasonRaw should be set")
	s.T().Logf("[TestBypassProposal] ExecutionError details: Transaction=%v RawRevert=%v Decoded=%q UnderlyingRaw=%q UnderlyingDecoded=%q",
		execErr.Transaction,
		execErr.RevertReasonRaw,
		execErr.RevertReasonDecoded,
		execErr.UnderlyingReasonRaw,
		execErr.UnderlyingReasonDecoded,
	)
	s.Equal(
		outOfBoundsGroupError.Name,
		execErr.UnderlyingReasonDecoded,
		"Decoded underlying reason should match MCMS error",
	)
	s.Equal(
		"CallReverted(truncated)",
		execErr.RevertReasonDecoded,
		"Decoded revert reason should capture CallReverted wrapper",
	)
}

// transferMCMSOwnershipToTimelock transfers ownership of MCMS from the current owner to the timelock contract.
// This is a two-step process with Ownable2Step:
// 1. Transfer ownership (initiated by current owner)
// 2. Accept ownership (must be called by the new owner, i.e., timelock)
func transferMCMSOwnershipToTimelock(
	t *testing.T,
	ctx context.Context,
	mcmsContract *bindings.ManyChainMultiSig,
	timelockAddr common.Address,
	currentOwnerAuth *bind.TransactOpts,
	client *ethclient.Client,
) {
	t.Helper()

	// Step 1: Transfer ownership from current owner to timelock
	tx, err := mcmsContract.TransferOwnership(currentOwnerAuth, timelockAddr)
	require.NoError(t, err, "Failed to transfer MCMS ownership to timelock")

	receipt, err := testutils.WaitMinedWithTxHash(ctx, client, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "TransferOwnership transaction failed")
}

// acceptMCMSOwnership accepts ownership of MCMS on behalf of the timelock.
// This creates a proposal that calls acceptOwnership() on MCMS and executes it through the timelock.
func acceptMCMSOwnership(
	t *testing.T,
	ctx context.Context,
	mcmsContract *bindings.ManyChainMultiSig,
	timelockContract *bindings.RBACTimelock,
	timelockAuth *bind.TransactOpts,
	client *ethclient.Client,
) {
	t.Helper()

	// Get the acceptOwnership method from MCMS ABI
	mcmsABI, err := bindings.ManyChainMultiSigMetaData.GetAbi()
	require.NoError(t, err, mcmsABIErrorMsg)

	method, exists := mcmsABI.Methods["acceptOwnership"]
	require.True(t, exists, "acceptOwnership method not found in MCMS ABI")

	// Pack the acceptOwnership call (no parameters, so just the method selector)
	calldata := method.ID

	// Create a bypass call to accept ownership
	// Since we're in test setup, we can use bypasserExecuteBatch directly
	calls := []bindings.RBACTimelockCall{
		{
			Target: mcmsContract.Address(),
			Value:  big.NewInt(0),
			Data:   calldata, // acceptOwnership has no parameters, so just the selector
		},
	}

	tx, err := timelockContract.BypasserExecuteBatch(timelockAuth, calls)
	require.NoError(t, err, "Failed to execute acceptOwnership through timelock")

	receipt, err := testutils.WaitMinedWithTxHash(ctx, client, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "AcceptOwnership transaction failed")
}

// createRevertingTransaction creates a transaction that will revert when executed.
// It calls SetConfig on the MCMS contract with invalid data (out-of-bounds group index)
// which will trigger an OutOfBoundsGroup error.
func createRevertingTransaction(t *testing.T, target common.Address) mcmtypes.Transaction {
	t.Helper()

	// Get the SetConfig method from the ABI
	mcmsABI, err := bindings.ManyChainMultiSigMetaData.GetAbi()
	require.NoError(t, err, mcmsABIErrorMsg)

	method, exists := mcmsABI.Methods["setConfig"]
	require.True(t, exists, "SetConfig method not found in ABI")

	// Create invalid SetConfig parameters that will cause a revert:
	// - signerAddresses: one valid address
	// - signerGroups: group index 255 (out of bounds, max is 31)
	// - groupQuorums: all zeros (invalid, but we'll use group index 255 which doesn't exist)
	// - groupParents: all zeros
	// - clearRoot: false
	signerAddresses := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}
	signerGroups := []uint8{255} // Out of bounds group index (max is 31)

	var groupQuorums [32]uint8
	var groupParents [32]uint8

	// Encode the parameters
	calldata, err := method.Inputs.Pack(
		signerAddresses,
		signerGroups,
		groupQuorums,
		groupParents,
		false, // clearRoot
	)
	require.NoError(t, err, "Failed to pack SetConfig parameters")

	// Prepend the method selector
	fullCalldata := append(method.ID, calldata...)

	return evm.NewTransaction(
		target,
		fullCalldata,
		big.NewInt(0),
		"",
		[]string{},
	)
}

// createScheduleTimelockProposal creates a TimelockProposal with schedule action.
// It configures the proposal for 1s delay.
func createScheduleTimelockProposal(
	t *testing.T,
	chainMetadata map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata,
	timelockAddresses map[mcmtypes.ChainSelector]string,
	batchOps []mcmtypes.BatchOperation,
) mcms.TimelockProposal {
	t.Helper()
	return mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Kind:                 mcmtypes.KindTimelockProposal,
			Description:          "Bypass proposal for error decoding test",
			ValidUntil:           2004259681,
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata:        chainMetadata,
		},
		Action:            mcmtypes.TimelockActionSchedule,
		Delay:             mcmtypes.MustParseDuration("1s"),
		TimelockAddresses: timelockAddresses,
		Operations:        batchOps,
	}
}

// createBypassTimelockProposal creates a TimelockProposal with bypass action.
// It configures the proposal for immediate bypass execution (no delay).
func createBypassTimelockProposal(
	t *testing.T,
	chainMetadata map[mcmtypes.ChainSelector]mcmtypes.ChainMetadata,
	timelockAddresses map[mcmtypes.ChainSelector]string,
	batchOps []mcmtypes.BatchOperation,
) mcms.TimelockProposal {
	t.Helper()
	return mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:              "v1",
			Kind:                 mcmtypes.KindTimelockProposal,
			Description:          "Bypass proposal for error decoding test",
			ValidUntil:           2004259681,
			OverridePreviousRoot: true,
			Signatures:           []mcmtypes.Signature{},
			ChainMetadata:        chainMetadata,
		},
		Action:            mcmtypes.TimelockActionBypass,
		Delay:             mcmtypes.MustParseDuration("0s"),
		TimelockAddresses: timelockAddresses,
		Operations:        batchOps,
	}
}

// convertTimelockProposal converts a TimelockProposal to a Proposal using the provided converters.
func convertTimelockProposal(
	t *testing.T,
	ctx context.Context,
	timelockProposal mcms.TimelockProposal,
	converters map[mcmtypes.ChainSelector]sdk.TimelockConverter,
) (mcms.Proposal, []common.Hash) {
	t.Helper()
	proposal, hashes, err := timelockProposal.Convert(ctx, converters)
	require.NoError(t, err)

	return proposal, hashes
}

// signAndValidateProposal signs and validates a proposal.
// It creates a signable object, signs it with the provided signers, validates configs, and verifies signatures meet quorum.
// proposal must be a pointer to ensure signatures are added to the same instance used by NewExecutable.
func signAndValidateProposal(
	t *testing.T,
	ctx context.Context,
	proposal *mcms.Proposal,
	inspectors map[mcmtypes.ChainSelector]sdk.Inspector,
	signerPrivateKeys []string,
) *mcms.Signable {
	t.Helper()
	signable, err := mcms.NewSignable(proposal, inspectors)
	require.NoError(t, err)

	err = signable.ValidateConfigs(ctx)
	require.NoError(t, err)

	// Sign with all provided signers
	// SignAndAppend automatically adds signatures to proposal.Signatures
	for _, privateKeyHex := range signerPrivateKeys {
		_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testutils.ParsePrivateKey(privateKeyHex)))
		require.NoError(t, err)
	}

	quorumMet, err := signable.ValidateSignatures(ctx)
	require.NoError(t, err)
	require.True(t, quorumMet, "quorum not met after signing")

	return signable
}

// setRootAndVerify sets the root on the MCMS contract and verifies it was set correctly.
func setRootAndVerify(
	t *testing.T,
	ctx context.Context,
	executable *mcms.Executable,
	chainSelector mcmtypes.ChainSelector,
	expectedRoot [32]byte,
	expectedValidUntil uint32,
	client *ethclient.Client,
	mcmsContract *bindings.ManyChainMultiSig,
) {
	t.Helper()

	tx, err := executable.SetRoot(ctx, chainSelector)
	require.NoError(t, err, "SetRoot failed")
	require.NotEmpty(t, tx.Hash, "SetRoot returned empty transaction hash")

	receipt, err := testutils.WaitMinedWithTxHash(ctx, client, common.HexToHash(tx.Hash))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "SetRoot transaction failed")

	root, err := mcmsContract.GetRoot(&bind.CallOpts{Context: ctx})
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root.Root, "root mismatch after SetRoot")
	require.Equal(t, expectedValidUntil, root.ValidUntil, "validUntil mismatch after SetRoot")
}

// assertExecutionError asserts that an error is an ExecutionError with expected fields.
// expectedSelector is optional - if provided (non-zero), it will assert that RevertReasonRaw.Selector matches.
func assertExecutionError(
	t *testing.T,
	err error,
	hasTransaction bool,
	hasUnderlyingReason bool,
	containsMessage string,
	expectedSelector [4]byte,
) *evm.ExecutionError {
	t.Helper()
	require.Error(t, err, "expected error but got nil")

	var execErr *evm.ExecutionError
	require.ErrorAs(t, err, &execErr, "error is not of type *evm.ExecutionError")

	if hasTransaction {
		require.NotNil(t, execErr.Transaction, "ExecutionError.Transaction is nil but expected to be set")
	}

	if hasUnderlyingReason {
		require.NotEmpty(t, execErr.UnderlyingReasonRaw, "ExecutionError.UnderlyingReasonRaw is empty but expected to be set")
	}

	if containsMessage != "" {
		errMsg := err.Error()
		require.Contains(t, errMsg, containsMessage, "error message does not contain expected text")
	}

	// Assert on selector if provided (non-zero selector)
	if expectedSelector != [4]byte{} {
		if execErr.RevertReasonRaw == nil {
			t.Logf("[assertExecutionError] Selector debug: expected=%#x but RevertReasonRaw is nil. OriginalErr=%v", expectedSelector, execErr.OriginalError)
		} else {
			t.Logf(
				"[assertExecutionError] Selector debug: expected=%#x actual=%#x rawHex=%s decoded=%s underlyingRaw=%s underlyingDecoded=%s original=%v",
				expectedSelector,
				execErr.RevertReasonRaw.Selector,
				common.Bytes2Hex(execErr.RevertReasonRaw.Combined()),
				execErr.RevertReasonDecoded,
				execErr.UnderlyingReasonRaw,
				execErr.UnderlyingReasonDecoded,
				execErr.OriginalError,
			)
		}
		require.NotNil(t, execErr.RevertReasonRaw, "RevertReasonRaw is nil but expected to be set")
		require.Equal(t, expectedSelector, execErr.RevertReasonRaw.Selector,
			"Error selector mismatch: expected 0x%x, got 0x%x", expectedSelector, execErr.RevertReasonRaw.Selector)
	}

	return execErr
}
