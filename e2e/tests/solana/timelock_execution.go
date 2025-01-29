//go:build e2e
// +build e2e

package solanae2e

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/access_controller"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	timelockutils "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/timelock"

	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testTimelockExecuteID = [32]byte{'t', 'e', 's', 't', '-', 'e', 'x', 'e', 'c', 't', 'i', 'm', 'e', 'l', 'o', 'c', 'k'}

const BatchAddAccessChunkSize = 24

// Test_Solana_TimelockExecute tests the timelock Execute functionality by scheduling a mint tokens transaction and
// executing it via the timelock ExecuteBatch
func (s *SolanaTestSuite) Test_Solana_TimelockExecute() {
	s.SetupTimelock(testTimelockExecuteID, 1)
	// Get required programs and accounts
	ctx := context.Background()
	timelock.SetProgramID(s.TimelockProgramID)
	access_controller.SetProgramID(s.AccessControllerProgramID)

	// Fund the auth private key
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	// Setup SPL token for testing a mint via timelock
	mintKeypair, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	mint := mintKeypair.PublicKey()
	// set up the token program
	signerPDA, err := mcmsSolana.FindTimelockSignerPDA(s.TimelockProgramID, testTimelockExecuteID)
	s.Require().NoError(err)
	receiverATA := s.setupTokenProgram(ctx, auth, signerPDA, mintKeypair)

	// Get receiverATA initial balance
	initialBalance, err := s.SolanaClient.GetTokenAccountBalance(
		context.Background(),
		receiverATA, // The associated token account address
		rpc.CommitmentProcessed,
	)
	s.Require().NoError(err)

	// Set propose roles
	proposerAndExecutorKey := s.setProposerAndExecutor(ctx, auth, s.Roles)
	s.Require().NotNil(proposerAndExecutorKey)

	// Schedule the mint tx
	var predecessor [32]byte
	salt := [32]byte{123}
	mintIx, operationID := s.scheduleMintTx(ctx,
		mint,
		receiverATA,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		signerPDA,
		*proposerAndExecutorKey,
		predecessor,
		salt)

	// --- act: call Timelock Execute ---
	executor := mcmsSolana.NewTimelockExecutor(s.SolanaClient, *proposerAndExecutorKey)
	contractID := mcmsSolana.ContractAddress(s.TimelockProgramID, testTimelockExecuteID)
	ixData, err := mintIx.Data()
	s.Require().NoError(err)
	accounts := mintIx.Accounts()
	accounts = append(accounts, &solana.AccountMeta{PublicKey: solana.Token2022ProgramID, IsSigner: false, IsWritable: false})
	solanaTx, err := mcmsSolana.NewTransaction(solana.Token2022ProgramID.String(), ixData, nil, accounts, "Token", []string{})
	s.Require().NoError(err)
	batchOp := types.BatchOperation{
		Transactions:  []types.Transaction{solanaTx},
		ChainSelector: s.ChainSelector,
	}
	// --- Wait for the operation to be ready ---
	s.waitForOperationToBeReady(ctx, testTimelockExecuteID, operationID)
	signature, err := executor.Execute(ctx, batchOp, contractID, predecessor, salt)
	s.Require().NoError(err)
	s.Require().NotEqual(signature, "")

	// --- assert balances
	finalBalance, err := s.SolanaClient.GetTokenAccountBalance(
		ctx,
		receiverATA,
		rpc.CommitmentProcessed,
	)
	s.Require().NoError(err)

	// final balance should be 1000000000000 more units
	s.Require().Equal(initialBalance.Value.Amount, "0")
	s.Require().Equal(finalBalance.Value.Amount, "1000000000000")
}

// setProposerAndExecutor sets the proposer for the timelock
func (s *SolanaTestSuite) setProposerAndExecutor(ctx context.Context, auth solana.PrivateKey, roleMap timelockutils.RoleMap) *solana.PrivateKey {
	proposerAndExecutorKey := solana.NewWallet()
	testutils.FundAccounts(ctx, []solana.PrivateKey{proposerAndExecutorKey.PrivateKey}, s.SolanaClient, s.T())

	// Add proposers to the timelock program
	batchAddAccessIxs, err := timelockutils.GetBatchAddAccessIxs(
		ctx,
		testTimelockExecuteID,
		roleMap[timelock.Proposer_Role].AccessController.PublicKey(),
		timelock.Proposer_Role,
		[]solana.PublicKey{proposerAndExecutorKey.PublicKey()},
		auth,
		BatchAddAccessChunkSize,
		s.SolanaClient)
	s.Require().NoError(err)
	for _, ix := range batchAddAccessIxs {
		testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, auth, rpc.CommitmentConfirmed)
	}

	// Add executor to the timelock program
	batchAddAccessIxs, err = timelockutils.GetBatchAddAccessIxs(
		ctx,
		testTimelockExecuteID,
		roleMap[timelock.Executor_Role].AccessController.PublicKey(),
		timelock.Executor_Role,
		[]solana.PublicKey{proposerAndExecutorKey.PublicKey()},
		auth,
		BatchAddAccessChunkSize,
		s.SolanaClient)
	s.Require().NoError(err)
	for _, ix := range batchAddAccessIxs {
		testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix}, auth, rpc.CommitmentConfirmed)
	}

	return &proposerAndExecutorKey.PrivateKey
}

// scheduleMintTx schedules a MintTx on the timelock
func (s *SolanaTestSuite) scheduleMintTx(
	ctx context.Context,
	mint,
	receiverATA, // The account that will receive the mint funds.
	roleAccessController solana.PublicKey, // Roles checker PDA account for checking roles

	// The account that will sign the transaction during timelock execution.
	// Note that this is a different account to the auth set for the timelock schedule ix,
	// this is because the timelock PDA signer account will sign the transaction during execution
	// and not the deployer account.
	authPublicKey solana.PublicKey,
	auth solana.PrivateKey, // The account to sign the init, append schedule instructions.
	predecessor, salt [32]byte) (instruction *token.Instruction, operationID [32]byte) {
	amount := 1000 * solana.LAMPORTS_PER_SOL
	mintIx, err := token.NewMintToInstruction(amount, mint, receiverATA, authPublicKey, nil).ValidateAndBuild()
	s.Require().NoError(err)
	for _, acc := range mintIx.Accounts() {
		if acc.PublicKey == authPublicKey {
			acc.IsSigner = false
		}
	}
	s.Require().NoError(err)
	// Get the operation ID
	ixData, err := mintIx.Data()
	s.Require().NoError(err)
	accounts := make([]timelock.InstructionAccount, 0, len(mintIx.Accounts())+1)
	for _, account := range mintIx.Accounts() {
		accounts = append(accounts, timelock.InstructionAccount{
			Pubkey:     account.PublicKey,
			IsSigner:   account.IsSigner,
			IsWritable: account.IsWritable,
		})
	}
	accounts = append(accounts, timelock.InstructionAccount{
		Pubkey:     solana.Token2022ProgramID,
		IsSigner:   false,
		IsWritable: false,
	})
	opInstructions := []timelock.InstructionData{{Data: ixData, ProgramId: solana.Token2022ProgramID, Accounts: accounts}}
	operationID, err = mcmsSolana.HashOperation(opInstructions, predecessor, salt)
	s.Require().NoError(err)
	operationPDA, err := mcmsSolana.FindTimelockOperationPDA(s.TimelockProgramID, testTimelockExecuteID, operationID)
	s.Require().NoError(err)
	configPDA, err := mcmsSolana.FindTimelockConfigPDA(s.TimelockProgramID, testTimelockExecuteID)
	s.Require().NoError(err)

	proposerAC := s.Roles[timelock.Proposer_Role].AccessController.PublicKey()

	// Preload and Init Operation
	ixs := []solana.Instruction{}
	initOpIx, err := timelock.NewInitializeOperationInstruction(
		testTimelockExecuteID,
		operationID,
		predecessor,
		salt,
		uint32(len(opInstructions)),
		operationPDA,
		configPDA,
		proposerAC,
		auth.PublicKey(),
		solana.SystemProgramID,
	).ValidateAndBuild()
	s.Require().NoError(err)
	ixs = append(ixs, initOpIx)
	// Append the ix

	for _, ix := range opInstructions {
		appendIx, errAppend := timelock.NewAppendInstructionsInstruction(
			testTimelockExecuteID,
			operationID,
			[]timelock.InstructionData{ix}, // this should be a slice of instruction within 1232 bytes
			operationPDA,
			configPDA,
			proposerAC,
			auth.PublicKey(),
			solana.SystemProgramID,
		).ValidateAndBuild()
		s.Require().NoError(errAppend)
		ixs = append(ixs, appendIx)
	}
	// Finalize Operation
	finOpIx, err := timelock.NewFinalizeOperationInstruction(
		testTimelockExecuteID,
		operationID,
		operationPDA,
		configPDA,
		proposerAC,
		auth.PublicKey(),
	).ValidateAndBuild()
	s.Require().NoError(err)
	ixs = append(ixs, finOpIx)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, ixs, auth, rpc.CommitmentConfirmed)
	// Schedule the operation
	scheduleIx, err := timelock.NewScheduleBatchInstruction(
		testTimelockExecuteID,
		operationID,
		1,
		operationPDA,
		configPDA,
		roleAccessController,
		auth.PublicKey(),
	).ValidateAndBuild()
	s.Require().NoError(err)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{scheduleIx}, auth, rpc.CommitmentConfirmed)

	return mintIx, operationID
}
