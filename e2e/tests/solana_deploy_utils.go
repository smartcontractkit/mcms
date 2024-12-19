package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	bpfloader "github.com/gagliardetto/solana-go/programs/bpf-loader"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

const MCMSBinPath = "e2e/artifacts/solana/mcm.so"

// CreateFundedTestAccount creates a new funded test account and waits for the transaction to be confirmed.
func CreateFundedTestAccount(ctx context.Context, client *rpc.Client) (solana.PrivateKey, error) {
	payer, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payer keypair: %w", err)
	}
	payerPublicKey := payer.PublicKey()
	airdropAmount := uint64(50_000_000_000)
	fmt.Printf("Requesting airdrop of %d lamports to payer: %s\n", airdropAmount, payerPublicKey)
	airdropTxSig, err := client.RequestAirdrop(
		ctx,
		payerPublicKey,
		airdropAmount,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return nil, fmt.Errorf("airdrop request failed: %w", err)
	}
	fmt.Printf("Airdrop transaction signature: %s\n", airdropTxSig)

	// Wait for the transaction to be confirmed
	if err := waitForConfirmation(ctx, client, airdropTxSig, 30*time.Second); err != nil {
		return nil, err
	}

	fmt.Println("Airdrop confirmed.")
	return payer, nil
}

// waitForConfirmation polls for a transaction's confirmation status until it's finalized or a timeout occurs.
func waitForConfirmation(ctx context.Context, client *rpc.Client, txSig solana.Signature, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timed out waiting for confirmation: %w", timeoutCtx.Err())
		case <-ticker.C:
			// Check the transaction status
			confirmation, err := client.GetSignatureStatuses(ctx, true, txSig)
			if err != nil {
				return fmt.Errorf("confirmation check failed: %w", err)
			}
			if confirmation != nil && confirmation.Value[0] != nil &&
				confirmation.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				return nil // Transaction confirmed
			}
		}
	}
}

// DeployProgramUsingBpfLoader deploys a program to Solana using the bpfloader library.
func DeployProgramUsingBpfLoader(
	ctx context.Context,
	rpcClient *rpc.Client,
	wsClient *ws.Client,
	payer *solana.PrivateKey,
	programKey *solana.PrivateKey,
	programData []byte,
) error {
	payerPubkey := payer.PublicKey()
	programPubkey := programKey.PublicKey()

	// Get the minimum balance for rent exemption
	minimumBalance, err := rpcClient.GetMinimumBalanceForRentExemption(
		ctx,
		uint64(len(programData)),
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return fmt.Errorf("failed to get minimum balance for rent exemption: %w", err)
	}

	// Use the Deploy function, assuming the program account does not exist
	initialBuilder, writeBuilders, finalBuilder, _, err := bpfloader.Deploy(
		payerPubkey,
		nil, // Assume program account does not exist
		programData,
		minimumBalance,
		solana.BPFLoaderProgramID,
		programPubkey,
		false, // Allow excessive balance?
	)
	if err != nil {
		return fmt.Errorf("failed to prepare deployment: %w", err)
	}

	// Send and confirm the initial transaction (if needed)
	if initialBuilder != nil {
		recentBlockhash, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return fmt.Errorf("failed to get latest blockhash: %w", err)
		}
		initialBuilder.SetRecentBlockHash(recentBlockhash.Value.Blockhash)
		tx, err := initialBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to build initial transaction: %w", err)
		}
		tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
			if key.Equals(payerPubkey) {
				return payer
			}
			if key.Equals(programPubkey) {
				return programKey
			}
			return nil
		})
		sig, err := confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
		if err != nil {
			return fmt.Errorf("failed to send and confirm initial transaction: %w", err)
		}
		fmt.Printf("Initial transaction confirmed with signature: %s\n", sig)
	}

	// Deploy program chunks in batches
	if err := deployInParallel(ctx, rpcClient, wsClient, writeBuilders, payer, programKey); err != nil {
		return fmt.Errorf("failed to deploy program chunks: %w", err)
	}

	// Send and confirm the final transaction
	if finalBuilder != nil {
		recentBlockhash, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return fmt.Errorf("failed to get latest blockhash: %w", err)
		}
		finalBuilder.SetRecentBlockHash(recentBlockhash.Value.Blockhash)
		tx, err := finalBuilder.Build()
		if err != nil {
			return fmt.Errorf("failed to build final transaction: %w", err)
		}
		tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
			if key.Equals(payerPubkey) {
				return payer
			}
			if key.Equals(programPubkey) {
				return programKey
			}
			return nil
		})
		sig, err := confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
		if err != nil {
			return fmt.Errorf("failed to send and confirm final transaction: %w", err)
		}
		fmt.Printf("Final transaction confirmed with signature: %s\n", sig)
	}

	fmt.Println("Program deployed successfully.")
	return nil
}

// deployInParallel sends all write transactions in parallel and waits for their confirmations.
func deployInParallel(
	ctx context.Context,
	rpcClient *rpc.Client,
	wsClient *ws.Client,
	txBuilders []*solana.TransactionBuilder,
	payer *solana.PrivateKey,
	programKey *solana.PrivateKey,
) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstError error

	// Channel to signal when all transactions are done
	done := make(chan struct{})

	for _, builder := range txBuilders {
		wg.Add(1)
		go func(builder *solana.TransactionBuilder) {
			defer wg.Done()

			// Fetch latest blockhash for the transaction
			recentBlockhash, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("failed to get latest blockhash: %w", err)
				}
				mu.Unlock()
				return
			}
			builder.SetRecentBlockHash(recentBlockhash.Value.Blockhash)

			// Build the transaction
			tx, err := builder.Build()
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("failed to build transaction: %w", err)
				}
				mu.Unlock()
				return
			}

			// Sign the transaction
			tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
				if key.Equals(payer.PublicKey()) {
					return payer
				}
				if key.Equals(programKey.PublicKey()) {
					return programKey
				}
				return nil
			})

			// Send and confirm the transaction
			txCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			_, err = confirm.SendAndConfirmTransaction(txCtx, rpcClient, wsClient, tx)
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("failed to send and confirm transaction: %w", err)
				}
				mu.Unlock()
				return
			}
		}(builder)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for all transactions or context timeout
	select {
	case <-done:
		// All goroutines completed
		return firstError
	case <-ctx.Done():
		// Context timeout or cancellation
		return fmt.Errorf("deployment interrupted: %w", ctx.Err())
	}
}
