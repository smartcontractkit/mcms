package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

// Configurer configures the MCM contract for EVM chains.
type Configurer struct {
	client ContractDeployBackend
	auth   *bind.TransactOpts
}

// NewConfigurer creates a new Configurer for EVM chains.
func NewConfigurer(client ContractDeployBackend, auth *bind.TransactOpts,
) *Configurer {
	return &Configurer{
		client: client,
		auth:   auth,
	}
}

// SetConfig sets the configuration for the MCM contract on the EVM chain.
func (c *Configurer) SetConfig(ctx context.Context, mcmAddr string, cfg *types.Config, clearRoot bool) (types.MinedTransaction, error) {
	mcmsC, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddr), c.client)
	if err != nil {
		return types.MinedTransaction{}, err
	}

	groupQuorums, groupParents, signerAddrs, signerGroups, err := ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.MinedTransaction{}, err
	}

	opts := *c.auth
	opts.Context = ctx

	tx, err := mcmsC.SetConfig(
		&opts,
		signerAddrs,
		signerGroups,
		groupQuorums,
		groupParents,
		clearRoot,
	)
	if err != nil {
		return types.MinedTransaction{}, err
	}

	return types.MinedTransaction{
		Hash: tx.Hash().Hex(),
		Tx:   tx,
	}, nil
}
