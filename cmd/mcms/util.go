package mcms

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	mcms_types "github.com/smartcontractkit/mcms/types"
)

func getMapKeys[T comparable, V any](m map[T]V) []T {
	keys := make([]T, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func loadPrivateKey() (*ecdsa.PrivateKey, error) {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		return nil, err
	}

	// Load PrivateKey
	pk := os.Getenv("PRIVATE_KEY")
	if pk == "" {
		return nil, errors.New("PRIVATE_KEY not found in .env file")
	}

	// Convert to ecdsa
	ecdsa, err := crypto.HexToECDSA(pk)
	if err != nil {
		return nil, err
	}

	return ecdsa, nil
}

// TODO: we shouldn't rely on the geth bind.DeployBackend here. This is some tech debt that needs to be resolved.
// The clients here will not always be EVM clients, so that needs to be abstracted away.
func loadRPCs(chainSelectors []mcms_types.ChainSelector) (map[mcms_types.ChainSelector]*ethclient.Client, error) {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		return nil, err
	}

	clients := make(map[mcms_types.ChainSelector]*ethclient.Client)
	for _, chainSelector := range chainSelectors {
		// Load RPC URL
		rpcKey := fmt.Sprintf("RPC_URL_%d", chainSelector)
		rpcURL := os.Getenv(rpcKey)
		if rpcURL == "" {
			return nil, errors.New(rpcKey + " not found in .env file")
		}

		// Connect to the Ethereum network
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			return nil, err
		}

		clients[chainSelector] = client
	}

	return clients, nil
}

// TODO: we shouldn't rely on the geth bind.DeployBackend here
// TODO: we should move this somewhere else
// func Confirm(ctx context.Context, selector chainsel.Chain, b bind.DeployBackend, txHash string) (*types.Receipt, error) {
// 	family, err := mcms_types.GetChainSelectorFamily(mcms_types.ChainSelector(selector.Selector))
// 	if err != nil {
// 		return nil, err
// 	}

// 	switch family {
// 	case chainsel.FamilyEVM:
// 		return EVMConfirm(ctx, b, txHash)
// 	default:
// 		return nil, errors.New("chain not supported")
// 	}
// }

func EVMConfirm(ctx context.Context, b bind.DeployBackend, txHash string) (*types.Receipt, error) {
	// Confirm a transaction on the EVM chain
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()

	logger := log.New("hash", txHash)
	for {
		receipt, err := b.TransactionReceipt(ctx, common.HexToHash(txHash))
		if err == nil {
			return receipt, nil
		}

		if errors.Is(err, ethereum.NotFound) {
			logger.Trace("Transaction not yet mined")
		} else {
			logger.Trace("Receipt retrieval failed", "err", err)
		}

		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}
