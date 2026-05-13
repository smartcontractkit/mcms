package stellar

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-stellar/bindings"
	stellarmcms "github.com/smartcontractkit/chainlink-stellar/bindings/contracts/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = (*Configurer)(nil)

// Configurer applies MCMS signer configuration on Stellar via Soroban set_config.
type Configurer struct {
	ConfigTransformer
	invoker bindings.Invoker
}

// NewConfigurer returns a Configurer that submits set_config through invoker.
func NewConfigurer(invoker bindings.Invoker) *Configurer {
	return &Configurer{invoker: invoker}
}

// SetConfig invokes set_config with signer address vec, group vec, and group tree bytes32 words.
func (c *Configurer) SetConfig(ctx context.Context, mcmAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	if cfg == nil {
		return types.TransactionResult{}, fmt.Errorf("nil config")
	}

	chainCfg, err := c.ToChainConfig(*cfg, nil)
	if err != nil {
		return types.TransactionResult{}, err
	}

	signerAddresses, signerGroups := setConfigVecsFromChainConfig(chainCfg)

	client, err := newMCMSClient(c.invoker, mcmAddr)
	if err != nil {
		return types.TransactionResult{}, err
	}

	if err := client.SetConfig(ctx, signerAddresses, signerGroups, chainCfg.GroupQuorums, chainCfg.GroupParents, clearRoot); err != nil {
		return types.TransactionResult{ChainFamily: chainsel.FamilyStellar}, err
	}

	return stellarTransactionResult(c.invoker), nil
}

func setConfigVecsFromChainConfig(chainCfg *stellarmcms.Config) (stellarmcms.SignerAddresses, stellarmcms.SignerGroups) {
	n := len(chainCfg.Signers)

	addrs := stellarmcms.SignerAddresses{Inner: make([][32]byte, n)}
	grps := stellarmcms.SignerGroups{Inner: make([]uint32, n)}

	for i := range chainCfg.Signers {
		addrs.Inner[i] = chainCfg.Signers[i].Addr
		grps.Inner[i] = chainCfg.Signers[i].Group
	}

	return addrs, grps
}
