//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	e2eutils "github.com/smartcontractkit/mcms/e2e/utils/solana"
	"github.com/smartcontractkit/mcms/sdk"
	solana2 "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testTimelockVerifyID = [32]byte{'t', 'e', 's', 't', '-', 'v', 'e', 'r', 'i', 'f', 'y'}

func (s *SolanaTestSuite) TestVerifyProgramViaTimelock() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	s.T().Cleanup(cancel)

	programID := s.AccessControllerProgramID.String()

	rpcURL := s.SolanaChain.Out.Nodes[0].HostHTTPUrl
	s.SetupMCM(testTimelockVerifyID)
	s.SetupTimelock(testTimelockVerifyID, 1*time.Second)

	// Auth key that controls the timelock roles
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	proposerAndExecutor, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)

	e2eutils.FundAccounts(s.T(), ctx, []solana.PublicKey{
		auth.PublicKey(), proposerAndExecutor.PublicKey(),
	}, 1, s.SolanaClient)
	mcmSignerPDA, err := solana2.FindSignerPDA(s.MCMProgramID, testTimelockVerifyID)
	s.Require().NoError(err)
	s.AssignRoleToAccounts(ctx, testTimelockVerifyID, auth, []solana.PublicKey{proposerAndExecutor.PublicKey(), mcmSignerPDA}, timelock.Proposer_Role)
	s.AssignRoleToAccounts(ctx, testTimelockVerifyID, auth, []solana.PublicKey{proposerAndExecutor.PublicKey(), mcmSignerPDA}, timelock.Executor_Role)

	// --- Step 1: export PDA tx using `solana-verify` CLI ---
	timelockSignerPDA, err := solana2.FindTimelockSignerPDA(s.TimelockProgramID, testTimelockVerifyID)
	require.NoError(s.T(), err)
	cmd := exec.CommandContext(ctx,
		"solana-verify",
		"export-pda-tx",
		"https://github.com/smartcontractkit/chainlink-ccip",
		"--program-id", programID,
		"--uploader", timelockSignerPDA.String(),
		"--url", rpcURL,
		"--encoding", "base58",
		"--compute-unit-price", "0",
		"--commit-hash", "07e5df862401c67c3b9c7e771a3abff64a111ac5",
		"--library-name", "access_controller",
	)

	out, err := cmd.CombinedOutput()
	s.Require().NoError(err, "solana-verify export-pda-tx failed: %s", string(out))

	lines := strings.Split(string(out), "\n")
	var base58EncodedTx string
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			base58EncodedTx = strings.TrimSpace(lines[i])
			break
		}
	}
	s.Require().NotEmpty(base58EncodedTx, "failed to extract base58-encoded transaction")

	fmt.Println("Base58 Encoded Tx:", base58EncodedTx)

	// --- Step 2: Decode transaction ---
	txBytes, err := base58.Decode(base58EncodedTx)
	s.Require().NoError(err, "failed to base58-decode exported tx")

	tx, err := solana.TransactionFromBytes(txBytes)
	s.Require().NoError(err)

	inst := tx.Message.Instructions[0]
	fmt.Println("number of instructions:", len(tx.Message.Instructions))
	resolved := resolveCompiledInstruction(s.T(), timelockSignerPDA, tx.Message, inst)
	fmt.Println("timelock accounts:")
	fmt.Println("timelock signer PDA:", timelockSignerPDA)
	fmt.Println("access controller program ID:", programID)
	fmt.Println("Resolved accounts:")
	for _, a := range resolved.Accounts() {
		fmt.Printf("- %s signer=%v writable=%v\n", a.PublicKey, a.IsSigner, a.IsWritable)
	}
	mcmsTx, err := solana2.NewTransaction(
		resolved.ProgID.String(),
		resolved.DataBytes,
		big.NewInt(0), resolved.Accounts(), "Verifier", []string{})
	require.NoError(s.T(), err, "failed to create transaction")
	batch := types.BatchOperation{
		Transactions:  []types.Transaction{mcmsTx},
		ChainSelector: s.ChainSelector,
	}
	validUntil := uint32(time.Now().Add(1 * time.Hour).Unix())

	timelockAddress := solana2.ContractAddress(s.TimelockProgramID, testTimelockVerifyID)
	mcmAddress := solana2.ContractAddress(s.MCMProgramID, testTimelockVerifyID)

	s.Require().NoError(err)
	testSigner := NewEVMTestAccount(s.T())
	configurer := solana2.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
	_, err = configurer.SetConfig(ctx, mcmAddress, &types.Config{
		Quorum:  1,
		Signers: []common.Address{testSigner.Address},
	}, true)
	s.Require().NoError(err)
	s.Require().NoError(err)

	e2eutils.FundAccounts(s.T(), ctx, []solana.PublicKey{mcmSignerPDA, timelockSignerPDA}, 1, s.SolanaClient)
	metadata, err := solana2.NewChainMetadata(
		uint64(0), // op count (starts at 0)
		s.MCMProgramID,
		testTimelockVerifyID,
		s.Roles[timelock.Proposer_Role].AccessController.PublicKey(),
		s.Roles[timelock.Canceller_Role].AccessController.PublicKey(),
		s.Roles[timelock.Executor_Role].AccessController.PublicKey(),
	)
	proposal, err := mcms.NewTimelockProposalBuilder().
		SetValidUntil(validUntil).
		SetVersion("v1").
		SetDescription("verify program via timelock").
		SetOverridePreviousRoot(true).
		AddTimelockAddress(s.ChainSelector, timelockAddress).
		AddChainMetadata(s.ChainSelector, metadata).
		SetAction(types.TimelockActionSchedule).
		AddOperation(batch).
		SetDelay(types.MustParseDuration("1s")).
		Build()
	s.Require().NoError(err)
	// Convert to mcms.Proposal
	converters := map[types.ChainSelector]sdk.TimelockConverter{
		s.ChainSelector: solana2.TimelockConverter{},
	}
	batch = proposal.Operations[0]
	gotProposal, gotPredecessors, err := proposal.Convert(ctx, converters)
	s.Require().NoError(err)
	s.Require().Equal([]common.Hash{mcms.ZERO_HASH}, gotPredecessors)

	// Sign proposal
	signable, err := mcms.NewSignable(&gotProposal, map[types.ChainSelector]sdk.Inspector{
		s.ChainSelector: solana2.NewInspector(s.SolanaClient),
	})
	s.Require().NoError(err)

	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(testSigner.PrivateKey))
	s.Require().NoError(err)

	// Set root + execute
	encoders, err := gotProposal.GetEncoders()
	s.Require().NoError(err)

	executor := solana2.NewExecutor(encoders[s.ChainSelector].(*solana2.Encoder), s.SolanaClient, proposerAndExecutor)
	execs := map[types.ChainSelector]sdk.Executor{s.ChainSelector: executor}

	executable, err := mcms.NewExecutable(&gotProposal, execs)
	s.Require().NoError(err)

	_, err = executable.SetRoot(ctx, s.ChainSelector)
	s.Require().NoError(err)

	for i := range gotProposal.Operations {
		_, err = executable.Execute(ctx, i)
		s.Require().NoError(err)
	}

	// Wait for operation to be ready
	salt := proposal.Salt()
	predecessor := gotPredecessors[0]
	ixData := getInstructionDataFromBatchOperation(s.T(), batch)
	opID, err := solana2.HashOperation(ixData, predecessor, salt)
	s.Require().NoError(err)
	s.waitForOperationToBeReady(ctx, testTimelockVerifyID, opID)

	// timelock execute
	timelockExec := solana2.NewTimelockExecutor(s.SolanaClient, proposerAndExecutor)
	sig, err := timelockExec.Execute(ctx, batch, timelockAddress, predecessor, salt)
	s.Require().NoError(err)

	s.T().Logf("Timelock execution completed. Signature: %s", sig)
}

func resolveCompiledInstruction(
	t *testing.T,
	timelockSignerPDA solana.PublicKey,
	msg solana.Message,
	compiled solana.CompiledInstruction,
) *solana.GenericInstruction {
	accounts := make(solana.AccountMetaSlice, len(compiled.Accounts))
	for i, idx := range compiled.Accounts {
		require.Less(t, int(idx), len(msg.AccountKeys), "account index out of range: %d", idx)
		pub := msg.AccountKeys[idx]
		isSigner := msg.IsSigner(pub)

		isWritable, err := msg.IsWritable(pub)
		require.NoError(t, err, "failed to check if account is writable")
		accounts[i] = &solana.AccountMeta{
			PublicKey:  pub,
			IsSigner:   isSigner,
			IsWritable: isWritable,
		}
	}
	require.Less(t, int(compiled.ProgramIDIndex), len(msg.AccountKeys), "program ID index out of range: %d", compiled.ProgramIDIndex)

	programID := msg.AccountKeys[compiled.ProgramIDIndex]

	data, err := base58.Decode(compiled.Data.String())
	if err != nil {
		fmt.Errorf("failed to decode instruction data: %w", err)
	}

	return &solana.GenericInstruction{
		ProgID:        programID,
		AccountValues: accounts,
		DataBytes:     data,
	}
}

func getInstructionDataFromBatchOperation(t *testing.T, batchOp types.BatchOperation) []bindings.InstructionData {
	instructionsData := make([]bindings.InstructionData, 0)
	for _, tx := range batchOp.Transactions {
		toProgramID, err := solana2.ParseProgramID(tx.To)
		require.NoError(t, err, "unable to parse program id from To field")

		var additionalFields solana2.AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			err = json.Unmarshal(tx.AdditionalFields, &additionalFields)
			require.NoError(t, err, "unable to unmarshal additional fields")
		}

		instructionsData = append(instructionsData, bindings.InstructionData{
			ProgramId: toProgramID,
			Data:      tx.Data,
			Accounts:  accountMetaToInstructionAccount(additionalFields.Accounts...),
		})
	}

	return instructionsData
}

func accountMetaToInstructionAccount(accounts ...*solana.AccountMeta) []bindings.InstructionAccount {
	instructionAccounts := make([]bindings.InstructionAccount, len(accounts))
	for i, account := range accounts {
		instructionAccounts[i] = bindings.InstructionAccount{
			Pubkey:     account.PublicKey,
			IsSigner:   account.IsSigner,
			IsWritable: account.IsWritable,
		}
	}

	return instructionAccounts
}
