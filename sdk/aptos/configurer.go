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
	role   TimelockRole

	bindingFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) mcms.MCMS

	skipSend bool
}

// NewConfigurer creates a new Configurer for an Aptos MCMS instance.
//
// options:
//
//	WithDoNotSendInstructionsOnChain: when selected, the Configurer instance will not
//			send the Aptos transaction to the blockchain but will return a prepared
//			MCMS types.Transaction instead.
func NewConfigurer(client aptos.AptosRpcClient, auth aptos.TransactionSigner, role TimelockRole, options ...configurerOption) *Configurer {
	c := &Configurer{
		client:    client,
		auth:      auth,
		role:      role,
		bindingFn: mcms.Bind,
	}
	for _, option := range options {
		option(c)
	}

	return c
}

type configurerOption func(*Configurer)

// WithDoNotSendInstructionsOnChain sets the configurer to not sign and send the configuration transaction
// but rather make it return a prepared MCMS types.Transaction instead.
// If set, the Hash field in the result will be empty.
func WithDoNotSendInstructionsOnChain() configurerOption {
	return func(c *Configurer) {
		c.skipSend = true
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

	if c.skipSend {
		moduleInfo, function, _, args, err := mcmsBinding.MCMS().Encoder().SetConfig(
			c.role.Byte(),
			signers,
			signerGroups,
			groupQuorum[:],
			groupParents[:],
			clearRoot,
		)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("encoding SetConfig call on Aptos mcms contract: %w", err)
		}
		tx, err := NewTransaction(
			moduleInfo.PackageName,
			moduleInfo.ModuleName,
			function,
			mcmsAddress,
			ArgsToData(args),
			"",
			nil,
		)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("creating mcms transaction: %w", err)
		}

		return types.TransactionResult{
			Hash:        "", // Returning no hash since the transaction hasn't been sent yet.
			ChainFamily: chain_selectors.FamilyAptos,
			RawData:     tx, // will be of type types.Transaction
		}, nil
	}

	tx, err := mcmsBinding.MCMS().SetConfig(
		opts,
		c.role.Byte(),
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
		RawData:     tx, // will be of type *api.PendingTransaction
	}, nil
}
