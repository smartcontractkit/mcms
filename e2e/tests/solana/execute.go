//go:build e2e
// +build e2e

package e2e_solana

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	mcmsSolana "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/types"
)

func (s *SolanaTestSuite) Test_Solana_Execute() {
	// --- arrange ---
	ctx := context.Background()
	mcmAddress := s.SolanaChain.SolanaPrograms["mcm"]
	mcmID := fmt.Sprintf("%s.%s", mcmAddress, testPDASeed)
	recipientAddress := s.SolanaChain.PublicKey
	recipetPubKey, err := solana.PublicKeyFromBase58(recipientAddress)
	s.Require().NoError(err)
	signerEVMAccount := NewEVMTestAccount(s.T())
	mcmConfig := types.Config{Quorum: 1, Signers: []common.Address{signerEVMAccount.Address}}

	auth, err := solana.PrivateKeyFromBase58(privateKey)
	s.Require().NoError(err)

	// Get the initial balance of the recipient
	initialBalance, err := s.SolanaClient.GetBalance(
		ctx,
		recipetPubKey,
		rpc.CommitmentProcessed,
	)
	s.Require().NoError(err)
	// Define transfer amount (e.g., 1 SOL = 1e9 lamports)
	transferAmount := uint64(1e9)
	// Create transfer instruction
	recent, err := s.SolanaClient.GetLatestBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		panic(err)
	}
	s.Require().NoError(err)
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				transferAmount,
				auth.PublicKey(),
				recipetPubKey,
			).Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(auth.PublicKey()),
	)
	s.Require().NoError(err)
	txBytes, err := tx.MarshalBinary()
	s.Require().NoError(err)
	solanaMcmTx, err := mcmsSolana.NewTransaction(mcmAddress,
		txBytes,
		[]solana.AccountMeta{
			{
				PublicKey:  auth.PublicKey(),
				IsSigner:   true,
				IsWritable: true,
			},
		},
		"test",
		[]string{"e2e-tests"},
	)
	s.Require().NoError(err)
	proposal, err := mcms.NewProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(10*time.Second).Unix())).
		SetDescription("proposal to test SetRoot").
		SetOverridePreviousRoot(true).
		AddChainMetadata(s.ChainSelector, types.ChainMetadata{MCMAddress: mcmID}).
		AddOperation(types.Operation{
			ChainSelector: s.ChainSelector,
			Transaction:   solanaMcmTx,
		}).
		Build()
	s.Require().NoError(err)

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

	// --- act: call Execute ---
	signature, err = executable.Execute(ctx, 0)
	s.Require().NoError(err)
	_, err = solana.SignatureFromBase58(signature)
	s.Require().NoError(err)

	// --- assert balances
	// Get the final balance of the recipient
	finalBalance, err := s.SolanaClient.GetBalance(
		ctx,
		recipetPubKey,
		rpc.CommitmentProcessed,
	)
	s.Require().NoError(err)
	// final balance should be 1 more SOL
	s.Require().Equal(initialBalance.Value+transferAmount, finalBalance.Value)
}
