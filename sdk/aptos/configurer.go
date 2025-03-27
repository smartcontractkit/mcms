package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS
}

func NewConfigurer(client aptos.AptosRpcClient, auth aptos.TransactionSigner) *Configurer {
	return &Configurer{
		client:    client,
		auth:      auth,
		bindingFn: mcms.Bind,
	}
}

func (c Configurer) SetConfig(ctx context.Context, mcmsAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddress, err)
	}
	mcmsBinding := c.bindingFn(mcmsAddress, c.client)
	opts := &bind.TransactOpts{Signer: c.auth}

	groupQuorum, groupParents, signerAddresses, signerGroups, err := evm.ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}
	signers := make([][]byte, len(signerAddresses))
	for i, addr := range signerAddresses {
		signers[i] = addr.Bytes()
	}

	tx, err := mcmsBinding.MCMS().SetConfig(
		opts,
		signers,
		signerGroups,
		groupQuorum[:],
		groupParents[:],
		clearRoot,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting config on Aptos mcms contract: %w", err)
	}
	return types.TransactionResult{
		Hash:        tx.Hash,
		ChainFamily: chain_selectors.FamilyAptos,
		RawData:     tx,
	}, nil
}
