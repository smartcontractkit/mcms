//go:build e2e

package tone2e

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ccip/bindings/common"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tlbe"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/wrappers"
)

const (
	EnvPathContracts = "PATH_CONTRACTS_TON"

	PathContractsMCMS     = "mcms.MCMS.compiled.json"
	PathContractsTimelock = "mcms.RBACTimelock.compiled.json"
)

func must[E any](out E, err error) E {
	if err != nil {
		panic(err)
	}

	return out
}

func LocalWalletDefault(client *ton.APIClient) (*wallet.Wallet, error) {
	walletVersion := wallet.HighloadV2Verified //nolint:staticcheck // only option in mylocalton-docker
	mcWallet, err := wallet.FromSeed(client, strings.Fields(blockchain.DefaultTonHlWalletMnemonic), walletVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet from seed: %w", err)
	}

	w := wallet.WithWorkchain(-1)
	mcFunderWallet, err := wallet.FromPrivateKeyWithOptions(client, mcWallet.PrivateKey(), walletVersion, w)
	if err != nil {
		return nil, fmt.Errorf("failed to create funder wallet from private key: %w", err)
	}

	// subwallet 42 has balance
	return mcFunderWallet.GetSubwallet(uint32(42))
}

func MCMSEmptyDataFrom(id uint32, owner *address.Address, chainID int64) mcms.Data {
	return mcms.Data{
		ID: id,
		Ownable: common.Ownable2Step{
			Owner:        owner,
			PendingOwner: nil,
		},
		Oracle:  tvm.ZeroAddress,
		Signers: must(tvm.MakeDict(map[*big.Int]mcms.Signer{}, 160)), // TODO: tvm.KeyUINT160
		Config: mcms.Config{
			Signers:      must(tvm.MakeDictFrom([]mcms.Signer{}, tvm.KeyUINT8)),
			GroupQuorums: must(tvm.MakeDictFrom([]mcms.GroupQuorum{}, tvm.KeyUINT8)),
			GroupParents: must(tvm.MakeDictFrom([]mcms.GroupParent{}, tvm.KeyUINT8)),
		},
		SeenSignedHashes: must(tvm.MakeDict(map[*big.Int]mcms.SeenSignedHash{}, tvm.KeyUINT256)),
		RootInfo: mcms.RootInfo{
			ExpiringRootAndOpCount: mcms.ExpiringRootAndOpCount{
				Root:       tlbe.NewUint256(big.NewInt(0)),
				ValidUntil: 0,
				OpCount:    0,
				OpPendingInfo: mcms.OpPendingInfo{
					ValidAfter:             0,
					OpFinalizationTimeout:  0,
					OpPendingReceiver:      tvm.ZeroAddress,
					OpPendingBodyTruncated: tlbe.NewUint256(big.NewInt(0)),
				},
			},
			RootMetadata: mcms.RootMetadata{
				ChainID:              big.NewInt(chainID),
				MultiSig:             tvm.ZeroAddress,
				PreOpCount:           0,
				PostOpCount:          0,
				OverridePreviousRoot: false,
			},
		},
	}
}

func TimelockEmptyDataFrom(id uint32) timelock.Data {
	return timelock.Data{
		ID:                       id,
		MinDelay:                 0,
		Timestamps:               cell.NewDict(256),
		BlockedFnSelectorsLen:    0,
		BlockedFnSelectors:       cell.NewDict(32),
		ExecutorRoleCheckEnabled: true,
		OpPendingInfo: timelock.OpPendingInfo{
			ValidAfter:            0,
			OpFinalizationTimeout: 0,
			OpPendingID:           tlbe.NewUint256(big.NewInt(0)),
		},
		RBAC: rbac.Data{
			Roles: cell.NewDict(256),
		},
	}
}

func DeployMCMSContract(ctx context.Context, client *ton.APIClient, w *wallet.Wallet, amount tlb.Coins, data mcms.Data) (*address.Address, error) {
	body := cell.BeginCell().EndCell() // empty cell, top up

	contractPath := filepath.Join(os.Getenv(EnvPathContracts), PathContractsMCMS)
	contractCode, err := wrappers.ParseCompiledContract(contractPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compiled contract: %w", err)
	}

	contractData, err := tlb.ToCell(data)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract data cell: %w", err)
	}

	_client := tracetracking.NewSignedAPIClient(client, *w)
	contract, _, err := wrappers.Deploy(ctx, &_client, contractCode, contractData, amount, body)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy contract: %w", err)
	}

	return contract.Address, nil
}

func DeployTimelockContract(ctx context.Context, client *ton.APIClient, w *wallet.Wallet, amount tlb.Coins, data timelock.Data, body timelock.Init) (*address.Address, error) {
	contractPath := filepath.Join(os.Getenv(EnvPathContracts), PathContractsTimelock)
	contractCode, err := wrappers.ParseCompiledContract(contractPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compiled contract: %w", err)
	}

	contractData, err := tlb.ToCell(data)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract data cell: %w", err)
	}

	bodyCell, err := tlb.ToCell(body)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract body cell: %w", err)
	}

	_client := tracetracking.NewSignedAPIClient(client, *w)
	contract, _, err := wrappers.Deploy(ctx, &_client, contractCode, contractData, amount, bodyCell)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy contract: %w", err)
	}

	return contract.Address, nil
}
