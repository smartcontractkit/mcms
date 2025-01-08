package solana

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = (*Inspector)(nil)

// Inspector is an Inspector implementation for Solana chains, giving access to the state of the MCMS contract
type Inspector struct {
	client *rpc.Client
}

// NewInspector creates a new Inspector for Solana chains
func NewInspector(client *rpc.Client) *Inspector {
	return &Inspector{client: client}
}

func (e *Inspector) GetConfig(ctx context.Context, address string) (*types.Config, error) {
	programID, instanceID, err := ParseContractAddress(address)
	if err != nil {
		return nil, err
	}

	configPDA, err := FindConfigPDA(programID, instanceID)
	if err != nil {
		return nil, err
	}

	var chainConfig bindings.MultisigConfig
	err = solanaCommon.GetAccountDataBorshInto(ctx, e.client, configPDA, rpc.CommitmentConfirmed, &chainConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to get config: %w", err)
	}

	mcmConfig, err := NewConfigTransformer().ToConfig(&chainConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to convert chain config: %w", err)
	}

	return mcmConfig, nil
}

func (e *Inspector) GetOpCount(ctx context.Context, mcmAddress string) (uint64, error) {
	programID, seed, err := ParseContractAddress(mcmAddress)
	if err != nil {
		return 0, err
	}
	pda, err := FindExpiringRootAndOpCountPDA(programID, seed)
	if err != nil {
		return 0, err
	}

	data, err := e.getExpiringRootAndOpCountData(ctx, pda)
	if err != nil {
		return 0, err
	}

	return data.OpCount, nil
}

func (e *Inspector) GetRoot(ctx context.Context, mcmAddress string) (common.Hash, uint32, error) {
	programID, seed, err := ParseContractAddress(mcmAddress)
	if err != nil {
		return common.Hash{}, 0, err
	}
	pda, err := FindExpiringRootAndOpCountPDA(programID, seed)
	if err != nil {
		return common.Hash{}, 0, err
	}

	data, err := e.getExpiringRootAndOpCountData(ctx, pda)
	if err != nil {
		return common.Hash{}, 0, err
	}

	return data.Root, data.ValidUntil, nil
}

func (e *Inspector) GetRootMetadata(ctx context.Context, mcmAddress string) (types.ChainMetadata, error) {
	programID, seed, err := ParseContractAddress(mcmAddress)
	if err != nil {
		return types.ChainMetadata{}, err
	}
	pda, err := FindRootMetadataPDA(programID, seed)
	if err != nil {
		return types.ChainMetadata{}, err
	}
	var newRootMetadata bindings.RootMetadata
	err = solanaCommon.GetAccountDataBorshInto(ctx, e.client, pda, rpc.CommitmentConfirmed, &newRootMetadata)
	if err != nil {
		return types.ChainMetadata{}, err
	}

	return types.ChainMetadata{
		StartingOpCount: newRootMetadata.PreOpCount,
		MCMAddress:      mcmAddress,
	}, nil
}

func (e *Inspector) getExpiringRootAndOpCountData(ctx context.Context, expiringRootAndOpCountPDA solana.PublicKey,
) (bindings.ExpiringRootAndOpCount, error) {
	var newRootAndOpCount bindings.ExpiringRootAndOpCount
	err := solanaCommon.GetAccountDataBorshInto(ctx, e.client, expiringRootAndOpCountPDA, rpc.CommitmentConfirmed,
		&newRootAndOpCount)
	if err != nil {
		return bindings.ExpiringRootAndOpCount{}, err
	}

	return newRootAndOpCount, nil
}
