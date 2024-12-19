package evm

import (
	"github.com/docker/docker/client"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ag_solanago "github.com/gagliardetto/solana-go"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer[SetConfigAccounts] = &Configurer{}

// Configurer configures the MCM contract for EVM chains.
type Configurer struct {
	client client.Client
	auth   *bind.TransactOpts
}

// NewConfigurer creates a new Configurer for EVM chains.
func NewConfigurer(client client.Client, auth *bind.TransactOpts,
) *Configurer {
	return &Configurer{
		client: client,
		auth:   auth,
	}
}

type SetConfigAccounts struct {
	MultiSigConfig ag_solanago.PublicKey
	ConfigSigners  ag_solanago.PublicKey
	Authority      ag_solanago.PublicKey
	SystemProgram  ag_solanago.PublicKey
}

// SetConfig sets the configuration for the MCM contract on the EVM chain.
func (c *Configurer) SetConfig(accounts SetConfigAccounts, cfg *types.Config, clearRoot bool) (string, error) {
	panic("implement me")
}
