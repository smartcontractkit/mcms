//go:build e2e

package solanae2e

import (
	"context"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"

	e2eutils "github.com/smartcontractkit/mcms/e2e/utils"
	e2esolanautils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testTimelockExecuteID = [32]byte{'t', 'e', 's', 't', '-', 'e', 'x', 'e', 'c', 't', 'i', 'm', 'e', 'l', 'o', 'c', 'k'}

const BatchAddAccessChunkSize = 24

// Test_Solana_TimelockExecute tests the timelock Execute functionality by scheduling a mint tokens transaction and
// executing it via the timelock ExecuteBatch
func (s *SolanaTestSuite) Test_Solana_TimelockExecute() {
	// --- arrange ---
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	s.T().Cleanup(cancel)

	token.SetProgramID(solana.Token2022ProgramID)
	s.SetupMCM(testTimelockExecuteID)
	s.SetupTimelock(testTimelockExecuteID, 1)

	// Create auth and executor private keys
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	proposers := generatePrivateKeys(s.T(), 5)
	proposerKey := e2eutils.Sample(proposers)
	executors := generatePrivateKeys(s.T(), 5)
	executorKey := e2eutils.Sample(executors)
	mintKey, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)

	accountsToFund := []solana.PublicKey{auth.PublicKey(), proposerKey.PublicKey(), executorKey.PublicKey()}
	e2esolanautils.FundAccounts(s.T(), accountsToFund, 1, s.SolanaClient)

	s.AssignRoleToAccounts(ctx, testTimelockExecuteID, auth, getPublicKeys(proposers), timelock.Proposer_Role)
	s.AssignRoleToAccounts(ctx, testTimelockExecuteID, auth, getPublicKeys(executors), timelock.Executor_Role)

	signerPDA, err := mcmsSolana.FindTimelockSignerPDA(s.TimelockProgramID, testTimelockExecuteID)
	s.Require().NoError(err)

	// Setup SPL token for testing a mint via timelock
	receiverATA := s.setupTokenProgram(ctx, auth, signerPDA, mintKey)

	predecessor := [32]byte{}
	salt := [32]byte{123}
	executor := mcmsSolana.NewTimelockExecutor(s.SolanaClient, executorKey)
	timelockAddress := mcmsSolana.ContractAddress(s.TimelockProgramID, testTimelockExecuteID)

	initialBalance := getBalance(ctx, s.T(), s.SolanaClient, receiverATA)

	// schedule mint transaction and build batch operation from the returned ScheduleBatch instruction
	mintIx, operationID := s.scheduleMintTx(ctx, mintKey.PublicKey(), receiverATA,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(), signerPDA, proposerKey,
		predecessor, salt)
	s.waitForOperationToBeReady(ctx, testTimelockExecuteID, operationID)

	// --- act: call Timelock Execute ---
	mcmsTransaction, err := mcmsSolana.NewTransactionFromInstruction(mintIx, "Token", nil)
	s.Require().NoError(err)
	batchOp := types.BatchOperation{Transactions: []types.Transaction{mcmsTransaction}, ChainSelector: s.ChainSelector}

	signature, err := executor.Execute(ctx, batchOp, timelockAddress, predecessor, salt)
	s.Require().NoError(err)
	s.Require().NotEmpty(signature)

	// --- assert ---
	finalBalance := getBalance(ctx, s.T(), s.SolanaClient, receiverATA)
	s.Require().Equal("0", initialBalance)
	s.Require().Equal("1000000000000", finalBalance)
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
	predecessor,
	salt [32]byte,
) (instruction *token.Instruction, operationID [32]byte) {
	amount := 1000 * solana.LAMPORTS_PER_SOL
	mintIx, err := token.NewMintToInstruction(amount, mint, receiverATA, authPublicKey, nil).ValidateAndBuild()
	s.Require().NoError(err)
	for _, acc := range mintIx.Accounts() {
		if acc.PublicKey == authPublicKey {
			acc.IsSigner = false
		}
	}

	// Get the operation ID
	ixData, err := mintIx.Data()
	s.Require().NoError(err)
	accounts := make([]timelock.InstructionAccount, len(mintIx.Accounts()))
	for i, account := range mintIx.Accounts() {
		accounts[i] = timelock.InstructionAccount{
			Pubkey:     account.PublicKey,
			IsSigner:   account.IsSigner,
			IsWritable: account.IsWritable,
		}
	}
	opInstructions := []timelock.InstructionData{{Data: ixData, ProgramId: mintIx.ProgramID(), Accounts: accounts}}

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

	for i, ixData := range opInstructions {
		initializeIx, errAppend := timelock.NewInitializeInstructionInstruction(
			testTimelockExecuteID,
			operationID,
			ixData.ProgramId,
			ixData.Accounts,
			operationPDA,
			configPDA,
			proposerAC,
			auth.PublicKey(),
			solana.SystemProgramID,
		).ValidateAndBuild()
		s.Require().NoError(errAppend)
		ixs = append(ixs, initializeIx)

		rawData := ixData.Data
		offset := 0

		for offset < len(rawData) {
			end := offset + mcmsSolana.AppendIxDataChunkSize()
			if end > len(rawData) {
				end = len(rawData)
			}
			chunk := rawData[offset:end]

			appendIx, appendErr := timelock.NewAppendInstructionDataInstruction(
				testTimelockExecuteID,
				operationID,
				uint32(i), // which instruction index we are chunking
				chunk,     // partial data
				operationPDA,
				configPDA,
				proposerAC,
				auth.PublicKey(),
				solana.SystemProgramID,
			).ValidateAndBuild()
			s.Require().NoError(appendErr)
			ixs = append(ixs, appendIx)
			offset = end
		}
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

func getBalance(ctx context.Context, t *testing.T, client *rpc.Client, account solana.PublicKey) string {
	t.Helper()

	balance, err := client.GetTokenAccountBalance(ctx, account, rpc.CommitmentProcessed)
	require.NoError(t, err)

	return balance.Value.Amount
}

func generatePrivateKeys(t *testing.T, count int) []solana.PrivateKey {
	t.Helper()

	return e2eutils.Times(count, func(_ int) solana.PrivateKey {
		key, err := solana.NewRandomPrivateKey()
		require.NoError(t, err)

		return key
	})
}

func getPublicKeys(privateKeys []solana.PrivateKey) []solana.PublicKey {
	publicKeys := make([]solana.PublicKey, len(privateKeys))
	for i, privateKey := range privateKeys {
		publicKeys[i] = privateKey.PublicKey()
	}

	return publicKeys
}
