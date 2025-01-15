//go:build e2e
// +build e2e

package solanae2e

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/testutils"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/tokens"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

var testPDASeedExec = [32]byte{'t', 'e', 's', 't', '-', 'e', 'x', 'e', 'c'}

func (s *SolanaTestSuite) Test_Solana_Execute() {
	// Get required programs and accounts
	ctx := context.Background()
	mcmAddress := s.SolanaChain.SolanaPrograms["mcm"]
	mcmID := fmt.Sprintf("%s.%s", mcmAddress, testPDASeedExec)
	signerEVMAccount := NewEVMTestAccount(s.T())
	signerPDA, err := mcmsSolana.FindSignerPDA(config.McmProgram, testPDASeedExec)
	s.Require().NoError(err)

	// Build a simple 1 signer config
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	// Fund the signer PDA account
	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)
	fundPDAIx, err := system.NewTransferInstruction(
		1*solana.LAMPORTS_PER_SOL,
		auth.PublicKey(),
		signerPDA,
	).ValidateAndBuild()
	s.Require().NoError(err)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{fundPDAIx}, auth, config.DefaultCommitment)

	// Setup SPL token for testing a mint via MCMS
	mintKeypair, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	mint := mintKeypair.PublicKey()
	tokenProgram := config.Token2022Program

	// Use CreateToken utility to get initialization instructions

	createTokenIxs, err := tokens.CreateToken(
		ctx,
		tokenProgram,     // token program
		mint,             // mint account
		auth.PublicKey(), // initial mint owner(admin)
		uint8(9),         // decimals
		s.SolanaClient,
		config.DefaultCommitment,
	)
	s.Require().NoError(err)
	authIx, err := tokens.SetTokenMintAuthority(tokenProgram, signerPDA, mint, auth.PublicKey())
	s.Require().NoError(err)
	setupIxs := append(createTokenIxs, authIx)
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, setupIxs, auth, config.DefaultCommitment, solanaCommon.AddSigners(mintKeypair))

	// Mint tokens to receiver
	receiver, err := solana.NewRandomPrivateKey()
	s.Require().NoError(err)
	ix1, receiverATA, err := tokens.CreateAssociatedTokenAccount(tokenProgram, mint, receiver.PublicKey(), auth.PublicKey())
	s.Require().NoError(err)
	s.Require().NotEqual(receiverATA.String(), "")
	testutils.SendAndConfirm(ctx, s.T(), s.SolanaClient, []solana.Instruction{ix1}, auth, config.DefaultCommitment, solanaCommon.AddSigners(mintKeypair))

	// Get receiverATA initial balance
	initialBalance, err := s.SolanaClient.GetTokenAccountBalance(
		context.Background(),
		receiverATA, // The associated token account address
		rpc.CommitmentProcessed,
	)
	s.Require().NoError(err)

	amount := 1000 * solana.LAMPORTS_PER_SOL
	ix2, err := token.NewMintToInstruction(amount, mint, receiverATA, signerPDA, nil).ValidateAndBuild()
	accounts := ix2.Accounts()
	for _, acc := range accounts {
		if acc.PublicKey == signerPDA {
			// important step: when using MCMS we want to set the signer to false because the rust contract will sign using invoke_signer
			// if we keep this true, we'll have errors requiring a private key for a PDA which is not possible.
			acc.IsSigner = false
		}
	}
	s.Require().NoError(err)
	ix2Bytes, err := ix2.Data()
	s.Require().NoError(err)

	// Build the mcms transaction for the proposal
	solanaMcmTxMint, err := mcmsSolana.NewTransaction(solana.Token2022ProgramID.String(),
		ix2Bytes,
		ix2.Accounts(),
		"Token",
		[]string{"minting-test"},
	)
	s.Require().NoError(err)

	// Create the proposal
	s.Require().NoError(err)
	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(10*time.Hour).Unix())).
		SetDescription("proposal to test Execute with a token distribution").
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.ChainSelector, types.ChainMetadata{MCMAddress: mcmID}).
		AddOperation(types.Operation{
			ChainSelector: s.ChainSelector,
			Transaction:   solanaMcmTxMint,
		}).
		Build()
	s.Require().NoError(err)

	// build encoders, executors and inspectors.
	encoders, err := proposal.GetEncoders()
	s.Require().NoError(err)
	encoder := encoders[s.ChainSelector].(*mcmsSolana.Encoder)
	executors := map[types.ChainSelector]sdk.Executor{s.ChainSelector: mcmsSolana.NewExecutor(s.SolanaClient, auth, encoder)}
	inspectors := map[types.ChainSelector]sdk.Inspector{s.ChainSelector: mcmsSolana.NewInspector(s.SolanaClient)}

	// sign proposal
	signable, err := mcms.NewSignable(proposal, inspectors)
	s.Require().NoError(err)
	s.Require().NotNil(signable)
	_, err = signable.SignAndAppend(mcms.NewPrivateKeySigner(signerEVMAccount.PrivateKey))
	s.Require().NoError(err)

	// set config
	configurer := mcmsSolana.NewConfigurer(s.SolanaClient, auth, s.ChainSelector)
	_, err = configurer.SetConfig(ctx, mcmID, &mcmConfig, true)
	s.Require().NoError(err)

	// --- act: call SetRoot ---
	executable, err := mcms.NewExecutable(proposal, executors)
	s.Require().NoError(err)
	signature, err := executable.SetRoot(ctx, s.ChainSelector)
	s.Require().NoError(err)
	_, err = solana.SignatureFromBase58(signature)
	s.Require().NoError(err)

	// --- act: call Execute ---
	signature, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	_, err = solana.SignatureFromBase58(signature)
	s.Require().NoError(err)

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
