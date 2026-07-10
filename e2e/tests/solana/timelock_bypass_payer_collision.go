//go:build e2e

package solanae2e

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"

	"github.com/smartcontractkit/mcms"
	e2eutils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	"github.com/smartcontractkit/mcms/sdk"
	solanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var (
	testPDASeedBypassPayerWith    = [32]byte{'t', 'e', 's', 't', '-', 'b', 'y', 'p', 'a', 's', 's', '-', 'p', 'a', 'y', 'e', 'r', '-', 'w'}
	testPDASeedBypassPayerWithout = [32]byte{'t', 'e', 's', 't', '-', 'b', 'y', 'p', 'a', 's', 's', '-', 'p', 'a', 'y', 'e', 'r', '-', 'n'}
)

const bypassPayerTransferLamports = 1_000_000 // 0.001 SOL

// TestBypassExecutePayerInRemainingAccounts covers the Solana bypass failure
// where the execute payer (the deployer key) also appears in the
// BypasserExecuteBatch op's remaining_accounts.
//
// Real-world shape: a BPF-loader `upgrade` instruction lists the deployer as the
// spill/close recipient — a writable, non-signer account. Off-chain the Solana
// converter forces every remaining account to IsSigner=false before computing
// the Merkle root, so the deployer is hashed with IsSigner=false. At execution
// time the same deployer key is the outer transaction fee payer, so the Solana
// runtime presents it to the MCM program as IsSigner=true. The MCM program
// rebuilds the Merkle leaf from the runtime account infos, hashes IsSigner=true,
// and the one-bit mismatch invalidates the proof -> ProofCannotBeVerified.
//
// This test uses a system.Transfer whose recipient is the deployer/executor
// wallet to reproduce the identical one-bit collision without deploying an
// upgradeable program + buffer.
//
//   - "with execute payers": the proposal is converted with
//     solanasdk.WithExecutePayers, so the converter marks the executor account
//     IsSigner=true before the root is computed and the bypass executes cleanly.
//   - "without execute payers": the same proposal converted without the context
//     override still fails with ProofCannotBeVerified, documenting the bug and
//     guarding against the fix silently becoming a no-op.
func (s *TestSuite) TestBypassExecutePayerInRemainingAccounts() {
	s.Run("with execute payers in context: bypass succeeds", func() {
		s.runBypassPayerCollision(testPDASeedBypassPayerWith, true)
	})
	s.Run("without execute payers in context: proof fails", func() {
		s.runBypassPayerCollision(testPDASeedBypassPayerWithout, false)
	})
}

// runBypassPayerCollision drives the full bypass flow (convert -> set config ->
// sign -> set root -> execute) for a batch whose inner instruction sends
// lamports to the executor wallet. When injectExecutePayer is true, the executor
// is threaded through context so the converter marks it as a signer and the
// bypass execute succeeds; otherwise the final op fails with
// ProofCannotBeVerified.
func (s *TestSuite) runBypassPayerCollision(seed [32]byte, injectExecutePayer bool) {
	// --- arrange ---
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	s.T().Cleanup(cancel)

	// wallet is the deployer key: MCM executor / outer transaction fee payer.
	wallet, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	s.SetupMCM(seed)
	s.SetupTimelock(seed, 1*time.Second)

	mcmSignerPDA, err := solanasdk.FindSignerPDA(s.MCMProgramID, seed)
	s.Require().NoError(err)
	// The MCM signer PDA drives the bypass instructions, so it must hold the
	// bypasser role.
	s.AssignRoleToAccounts(ctx, seed, wallet, []solana.PublicKey{mcmSignerPDA}, timelock.Bypasser_Role)

	timelockSignerPDA, err := solanasdk.FindTimelockSignerPDA(s.TimelockProgramID, seed)
	s.Require().NoError(err)

	// Fund the timelock signer PDA (transfer source, signs via CPI) and the mcm
	// signer PDA.
	e2eutils.FundAccounts(s.T(), []solana.PublicKey{mcmSignerPDA, timelockSignerPDA}, 1, s.SolanaClient)

	mcmAddress := solanasdk.ContractAddress(s.MCMProgramID, seed)
	timelockAddress := solanasdk.ContractAddress(s.TimelockProgramID, seed)

	// --- inner "spill-like" instruction ---
	// Transfer lamports from the timelock signer PDA to the deployer wallet.
	// The recipient (wallet) is a writable, non-signer account: exactly the
	// role a BPF upgrade spill account plays in production.
	transferIx, err := system.NewTransferInstruction(bypassPayerTransferLamports, timelockSignerPDA, wallet.PublicKey()).
		ValidateAndBuild()
	s.Require().NoError(err)

	transferTx, err := solanasdk.NewTransactionFromInstruction(transferIx, "System",
		[]string{"bypass-payer-collision"})
	s.Require().NoError(err)

	batchOp := types.BatchOperation{
		ChainSelector: s.ChainSelector,
		Transactions:  []types.Transaction{transferTx},
	}

	// --- chain metadata ---
	opCount, err := solanasdk.NewInspector(s.SolanaClient).GetOpCount(ctx, mcmAddress)
	s.Require().NoError(err)
	metadata, err := solanasdk.NewChainMetadata(opCount, s.MCMProgramID, seed,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Bypasser_Role].AccessController.PublicKey())
	s.Require().NoError(err)

	// --- bypass proposal ---
	timelockProposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(2051222400). // 2035-01-01T12:00:00 UTC
		SetDescription("bypass proposal: executor payer appears in remaining_accounts").
		SetOverridePreviousRoot(true).
		SetDelay(types.NewDuration(1*time.Second)).
		SetAction(types.TimelockActionBypass).
		AddTimelockAddress(s.ChainSelector, timelockAddress).
		AddChainMetadata(s.ChainSelector, metadata).
		AddOperation(batchOp).
		Build() //nolint:contextcheck //OPT-400
	s.Require().NoError(err)

	converters := map[types.ChainSelector]sdk.TimelockConverter{
		s.ChainSelector: solanasdk.TimelockConverter{},
	}

	// The fix: when the expected execute payer is threaded through context, the
	// converter marks that account IsSigner=true so the off-chain Merkle root
	// matches what the runtime presents at execution time.
	if injectExecutePayer {
		ctx = solanasdk.WithExecutePayers(ctx, solanasdk.ExecutePayers{
			s.ChainSelector: wallet.PublicKey(),
		})
	}

	mcmsProposal, _, err := timelockProposal.Convert(ctx, converters)
	s.Require().NoError(err)

	// The executor wallet lands in the final BypasserExecuteBatch op as a
	// writable remaining account. Its IsSigner flag must reflect whether the
	// execute-payer override was applied.
	s.assertExecutorSignerBit(mcmsProposal, wallet.PublicKey(), injectExecutePayer)

	// --- set config + sign + set root ---
	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}
	configurer := solanasdk.NewConfigurer(s.SolanaClient, wallet, s.ChainSelector)
	_, err = configurer.SetConfig(ctx, mcmAddress, &mcmConfig, true)
	s.Require().NoError(err)

	inspectors := map[types.ChainSelector]sdk.Inspector{s.ChainSelector: solanasdk.NewInspector(s.SolanaClient)}
	signable, err := mcms.NewSignable(&mcmsProposal, inspectors) //nolint:contextcheck //OPT-400
	s.Require().NoError(err)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerEVMAccount.PrivateKey)) //nolint:contextcheck //OPT-400
	s.Require().NoError(err)

	encoders, err := mcmsProposal.GetEncoders() //nolint:contextcheck //OPT-400
	s.Require().NoError(err)
	encoder := encoders[s.ChainSelector].(*solanasdk.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{
		s.ChainSelector: solanasdk.NewExecutor(encoder, s.SolanaClient, wallet),
	}
	executable, err := mcms.NewExecutable(&mcmsProposal, executors) //nolint:contextcheck //OPT-400
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.ChainSelector)
	s.Require().NoError(err)

	// --- act + assert ---
	// The set-up ops (init/append/finalize bypasser operation) never include the
	// executor key, so their proofs verify regardless. Only the final
	// BypasserExecuteBatch op carries the executor key in its remaining accounts.
	lastOp := len(mcmsProposal.Operations) - 1
	s.Require().Positive(lastOp, "expected multiple bypass ops")

	balanceBefore := s.lamports(ctx, timelockSignerPDA)

	for i := range mcmsProposal.Operations {
		_, execErr := executable.Execute(ctx, i)
		switch {
		case i < lastOp:
			s.Require().NoError(execErr, "unexpected failure on op %d before BypasserExecuteBatch", i)
		case injectExecutePayer:
			s.Require().NoError(execErr, "BypasserExecuteBatch should succeed once the execute payer is a signer")
		default:
			s.Require().Error(execErr, "expected BypasserExecuteBatch to fail due to execute-payer signer collision")
			s.Require().ErrorContains(execErr, "ProofCannotBeVerified")
		}
	}

	if injectExecutePayer {
		// The inner transfer actually moved lamports out of the timelock signer PDA.
		balanceAfter := s.lamports(ctx, timelockSignerPDA)
		s.Require().Equal(balanceBefore-bypassPayerTransferLamports, balanceAfter,
			"timelock signer PDA should have sent exactly the transfer amount")
	}
}

// assertExecutorSignerBit checks the executor key appears in the last converted
// op (BypasserExecuteBatch) as a writable remaining account with the expected
// IsSigner flag.
func (s *TestSuite) assertExecutorSignerBit(proposal mcms.Proposal, executor solana.PublicKey, wantSigner bool) {
	s.Require().NotEmpty(proposal.Operations)
	lastOp := proposal.Operations[len(proposal.Operations)-1]

	var fields solanasdk.AdditionalFields
	s.Require().NoError(json.Unmarshal(lastOp.Transaction.AdditionalFields, &fields))

	found := false
	for _, acc := range fields.Accounts {
		if acc.PublicKey.Equals(executor) {
			found = true
			s.Require().Equal(wantSigner, acc.IsSigner, "executor IsSigner flag mismatch in converted bypass op")
			s.Require().True(acc.IsWritable, "executor (transfer recipient) should be writable")
		}
	}
	s.Require().True(found, "executor key must appear in the BypasserExecuteBatch remaining accounts")
}

// lamports returns the current lamport balance of the given account.
func (s *TestSuite) lamports(ctx context.Context, account solana.PublicKey) uint64 {
	res, err := s.SolanaClient.GetBalance(ctx, account, rpc.CommitmentConfirmed)
	s.Require().NoError(err)

	return res.Value
}
