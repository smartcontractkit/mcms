package evm

import (
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
func (c *Configurer) SetConfig(mcmAddr string, cfg *types.Config, clearRoot bool) (string, error) {
	mcmsC, err := bindings.NewManyChainMultiSig(common.HexToAddress(mcmAddr), c.client)
	if err != nil {
		return "", err
	}

	groupQuorums, groupParents, signerAddrs, signerGroups, err := extractSetConfigInputs(cfg)
	if err != nil {
		return "", err
	}

	tx, err := mcmsC.SetConfig(
		c.auth,
		signerAddrs,
		signerGroups,
		groupQuorums,
		groupParents,
		clearRoot,
	)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}
