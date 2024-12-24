package solana

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/mcms"

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

func (e *Inspector) GetConfig(mcmAddress string) (*types.Config, error) {
	ctx, cancel := context.WithCancel(context.Background()) // FIXME: add context as a method parameter?
	defer cancel()

	configPDA := mcms.McmConfigAddress(mcmName)

	var chainConfig mcm.MultisigConfig
	err := solanaCommon.GetAccountDataBorshInto(ctx, e.client, configPDA, rpc.CommitmentConfirmed, &chainConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to get config: %w", err)
	}

	config, err := NewConfigTransformer().ToConfig(&chainConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to convert chain config: %w", err)
	}

	return config, nil
}

func (e *Inspector) GetOpCount(mcmAddress string) (uint64, error) {
	return 0, nil
}

func (e *Inspector) GetRoot(mcmAddress string) (common.Hash, uint32, error) {
	return common.Hash{}, 0, nil
}

func (e *Inspector) GetRootMetadata(mcmAddress string) (types.ChainMetadata, error) {
	return types.ChainMetadata{}, nil
}
