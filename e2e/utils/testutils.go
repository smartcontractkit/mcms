package testutils

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
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

func ReadFixture(path string) (*os.File, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("Failed to get current file path")
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	fixturePath := filepath.Join(projectRoot, "e2e", "fixtures", path)

	file, err := os.Open(fixturePath)

	if err != nil {
		return nil, fmt.Errorf("Failed to open fixture file: %w", err)
	}

	return file, nil
}

// source: https://github.com/samber/lo/blob/49f24de9198ce4500df6cbef3260066bf777da74/find.go#L610-L613
func Sample[T any](collection []T) T {
	size := len(collection)
	if size == 0 {
		var empty T
		return empty
	}

	return collection[rand.IntN(size)]
}

// source: https://github.com/samber/lo/blob/49f24de9198ce4500df6cbef3260066bf777da74/slice.go#L126
func Times[T any](count int, iteratee func(index int) T) []T {
	result := make([]T, count)
	for i := range count {
		result[i] = iteratee(i)
	}

	return result
}
