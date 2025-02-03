package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/ethereum/go-ethereum/common"

	aptosutil "github.com/smartcontractkit/mcms/e2e/utils/aptos"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Inspector = &Inspector{}

type Inspector struct {
	ConfigTransformer
	client *aptos.NodeClient
}

func NewInspector(client *aptos.NodeClient) *Inspector {
	return &Inspector{client: client}
}

func (i Inspector) GetConfig(ctx context.Context, mcmAddr string) (*types.Config, error) {
	payload, err := aptosutil.BuildViewPayload(
		mcmAddr+"::mcms::get_config",
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}
	data, err := i.client.View(payload)
	if err != nil {
		return nil, fmt.Errorf("read: mcms::get_config: %w", err)
	}

	var (
		response ManyChainMultiSigConfig
	)

	if err := aptosutil.DecodeAptosJsonValue(data, &response); err != nil {
		return nil, fmt.Errorf("decode: mcms::get_config: %w", err)
	}

	return i.ToConfig(response)
}

func (i Inspector) GetOpCount(ctx context.Context, mcmAddr string) (uint64, error) {
	payload, err := aptosutil.BuildViewPayload(
		mcmAddr+"::mcms::get_op_count",
		nil,
		nil,
		nil,
	)
	if err != nil {
		return 0, err
	}
	data, err := i.client.View(payload)
	if err != nil {
		return 0, fmt.Errorf("read: mcms::get_op_count: %w", err)
	}

	var (
		opcount uint64
	)

	if err := aptosutil.DecodeAptosJsonValue(data, &opcount); err != nil {
		return 0, fmt.Errorf("decode: mcms::get_op_count: %w", err)
	}

	return opcount, nil
}

func (i Inspector) GetRoot(ctx context.Context, mcmAddr string) (common.Hash, uint32, error) {
	payload, err := aptosutil.BuildViewPayload(
		mcmAddr+"::mcms::get_root",
		nil,
		nil,
		nil,
	)
	if err != nil {
		return common.Hash{}, 0, err
	}
	data, err := i.client.View(payload)
	if err != nil {
		return common.Hash{}, 0, fmt.Errorf("read: mcms::get_root: %w", err)
	}

	var (
		hash       []byte
		validUntil uint32
	)

	if err := aptosutil.DecodeAptosJsonValue(data, &hash, &validUntil); err != nil {
		return common.Hash{}, 0, fmt.Errorf("decode: mcms::get_root: %w", err)
	}

	return common.BytesToHash(hash), validUntil, nil
}

func (i Inspector) GetRootMetadata(ctx context.Context, mcmAddr string) (types.ChainMetadata, error) {
	payload, err := aptosutil.BuildViewPayload(
		mcmAddr+"::mcms::get_root_metadata",
		nil,
		nil,
		nil,
	)
	if err != nil {
		return types.ChainMetadata{}, err
	}
	data, err := i.client.View(payload)
	if err != nil {
		return types.ChainMetadata{}, fmt.Errorf("read: mcms::get_root_metadata: %w", err)
	}

	var (
		metadata ManyChainMultiSigRootMetadata
	)

	if err := aptosutil.DecodeAptosJsonValue(data, &metadata); err != nil {
		return types.ChainMetadata{}, fmt.Errorf("decode: mcms::get_root_metadata: %w", err)
	}

	return types.ChainMetadata{
		StartingOpCount: metadata.PreOpCount,
		MCMAddress:      metadata.Multisig,
	}, nil
}
