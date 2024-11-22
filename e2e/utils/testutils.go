package testutils

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ParsePrivateKey converts a private key string into an *ecdsa.PrivateKey.
func ParsePrivateKey(privateKeyHex string) *ecdsa.PrivateKey {
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // Remove "0x" prefix
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	return privateKey
}

// WaitMinedWithTxHash waits for a transaction to be mined on the blockchain using its transaction hash.
// It stops waiting when the context is canceled.
func WaitMinedWithTxHash(ctx context.Context, b bind.DeployBackend, txHash common.Hash) (*types.Receipt, error) {
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()

	log.Printf("Waiting for transaction to be mined: %s", txHash.Hex())

	for {
		// Try to fetch the receipt
		receipt, err := b.TransactionReceipt(ctx, txHash)
		if err == nil {
			log.Printf("Transaction mined successfully: %s", txHash.Hex())
			return receipt, nil
		}

		if errors.Is(err, ethereum.NotFound) {
			// Transaction not mined yet
			log.Printf("Transaction not yet mined: %s", txHash.Hex())
		} else {
			// Log the error but keep trying
			log.Printf("Error retrieving receipt for transaction %s: %v", txHash.Hex(), err)
		}

		// Wait for the next round or context cancellation
		select {
		case <-ctx.Done():
			// Context canceled or timeout reached
			return nil, ctx.Err()
		case <-queryTicker.C:
			// Continue polling
		}
	}
}
