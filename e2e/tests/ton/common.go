//go:build e2e

package tone2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tracetracking"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/tvm"
	"github.com/smartcontractkit/chainlink-ton/pkg/ton/wrappers"

	"github.com/smartcontractkit/mcms/internal/testutils"
	"github.com/smartcontractkit/mcms/types"
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

type DeployOpts struct {
	// Connection
	Client *ton.APIClient
	Wallet *wallet.Wallet

	// Deployment info
	ContractPath string

	Amount tlb.Coins
	Data   any
	Body   any
}

func DeployContract(ctx context.Context, opts DeployOpts) (*address.Address, error) {
	contractCode, err := wrappers.ParseCompiledContract(opts.ContractPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compiled contract: %w", err)
	}

	contractData, ok := opts.Data.(*cell.Cell) // Cell or we try to decode
	if !ok {
		contractData, err = tlb.ToCell(opts.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to create contract data cell: %w", err)
		}
	}

	bodyCell, ok := opts.Body.(*cell.Cell) // Cell or we try to decode
	if !ok {
		bodyCell, err = tlb.ToCell(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create contract body cell: %w", err)
		}
	}

	_client := tracetracking.NewSignedAPIClient(opts.Client, *opts.Wallet)
	contract, _, err := wrappers.Deploy(ctx, &_client, contractCode, contractData, opts.Amount, bodyCell)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy contract: %w", err)
	}

	return contract.Address, nil
}

func NewInitializedAddress(ctx context.Context, s suite.Suite, tonClient *ton.APIClient, w *wallet.Wallet) *address.Address {
	walletA, err := tvm.NewRandomV5R1TestWallet(tonClient, -217)
	s.Require().NoError(err)
	// Fund wallet
	signedClient := tracetracking.NewSignedAPIClient(tonClient, *w)
	_, err = signedClient.SendAndWaitForTrace(ctx, *walletA.WalletAddress(),
		&wallet.Message{
			Mode: wallet.PayGasSeparately,
			InternalMessage: &tlb.InternalMessage{
				IHRDisabled: true,
				Bounce:      false,
				DstAddr:     walletA.WalletAddress(),
				Amount:      tlb.MustFromTON("0.1"),
				Body:        nil,
			},
		})
	s.Require().NoError(err)

	// Init wallet
	newSignedClient := tracetracking.NewSignedAPIClient(tonClient, *walletA)
	_, err = newSignedClient.SendAndWaitForTrace(ctx, *walletA.WalletAddress(), &wallet.Message{
		Mode: wallet.PayGasSeparately,
		InternalMessage: &tlb.InternalMessage{
			IHRDisabled: true,
			Bounce:      false,
			DstAddr:     walletA.WalletAddress(),
			Amount:      tlb.MustFromTON("0.1"),
			Body:        nil,
		},
	})
	s.Require().NoError(err)
	return walletA.WalletAddress().Bounce(true)
}

func DeployMCMSContract(ctx context.Context, client *ton.APIClient, w *wallet.Wallet, amount tlb.Coins, data mcms.Data) (*address.Address, error) {
	return DeployContract(ctx, DeployOpts{
		Client:       client,
		Wallet:       w,
		ContractPath: filepath.Join(os.Getenv(EnvPathContracts), PathContractsMCMS),
		Amount:       amount,
		Data:         data,
		Body:         cell.BeginCell().EndCell(), // empty cell, top up
	})
}

func DeployTimelockContract(ctx context.Context, client *ton.APIClient, w *wallet.Wallet, amount tlb.Coins, data timelock.Data, body timelock.Init) (*address.Address, error) {
	return DeployContract(ctx, DeployOpts{
		Client:       client,
		Wallet:       w,
		ContractPath: filepath.Join(os.Getenv(EnvPathContracts), PathContractsTimelock),
		Amount:       amount,
		Data:         data,
		Body:         body,
	})
}

// GenSimpleTestMCMSConfig generates a simple test configuration that's used in e2e tests.
func GenSimpleTestMCMSConfig(signers []testutils.ECDSASigner) *types.Config {
	return &types.Config{
		Quorum:  1,
		Signers: []common.Address{signers[0].Address()},
		GroupSigners: []types.Config{
			{
				Quorum:       1,
				Signers:      []common.Address{signers[1].Address()},
				GroupSigners: []types.Config{},
			},
		},
	}
}
