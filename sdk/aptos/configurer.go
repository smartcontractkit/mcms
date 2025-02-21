package aptos

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-internal-integrations/aptos/bindings/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

const (
	// TODO this is a const in the mcms.move contract (MAX_NUM_SIGNERS), make it available somewhere
	MAX_NUM_SIGNERS = 200
)

var _ sdk.Configurer = &Configurer{}

type Configurer struct {
	client aptos.AptosRpcClient
	auth   aptos.TransactionSigner
}

func NewConfigurer(client aptos.AptosRpcClient, auth aptos.TransactionSigner) *Configurer {
	return &Configurer{
		client: client,
		auth:   auth,
	}
}

func (c Configurer) SetConfig(ctx context.Context, mcmAddr string, cfg *types.Config, clearRoot bool) (types.TransactionResult, error) {
	mcmsAddress := aptos.AccountAddress{}
	_ = mcmsAddress.ParseStringRelaxed(mcmAddr)
	mcmsC := mcms.Bind(mcmsAddress, c.client)
	opts := &bind.TransactOpts{Signer: c.auth}

	groupQuorum, groupParents, signerAddresses, signerGroups, err := evm.ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}
	if len(signerAddresses) > MAX_NUM_SIGNERS {
		return types.TransactionResult{}, fmt.Errorf("too many signers (max %v)", MAX_NUM_SIGNERS)
	}

	tx, err := mcmsC.MCMS.SetConfig(
		opts,
		signerAddresses,
		signerGroups,
		groupQuorum[:],
		groupParents[:],
		clearRoot,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("setting config on Aptos mcms contract: %w", err)
	}
	return types.TransactionResult{
		Hash:           tx.Hash,
		ChainFamily:    chain_selectors.FamilyAptos,
		RawTransaction: tx,
	}, nil
}
