package aptos

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/aptos-labs/aptos-go-sdk"

	"github.com/smartcontractkit/chainlink-aptos/bindings/bind"
	"github.com/smartcontractkit/chainlink-aptos/bindings/curse_mcms"
	"github.com/smartcontractkit/chainlink-aptos/bindings/mcms"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

// TimelockConfigurer configures timelock parameters on Aptos chains.
// UpdateDelay returns a prepared MCMS transaction instead of executing on-chain.
type TimelockConfigurer struct {
	client    aptos.AptosRpcClient
	mcmsType  MCMSType
	encoderFn func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockUpdateDelayEncoder
}

type timelockUpdateDelayEncoder interface {
	TimelockUpdateMinDelay(newMinDelay uint64) (bind.ModuleInformation, string, []aptos.TypeTag, [][]byte, error)
}

// NewTimelockConfigurer creates a new TimelockConfigurer for Aptos chains.
func NewTimelockConfigurer(client aptos.AptosRpcClient) *TimelockConfigurer {
	return NewTimelockConfigurerWithMCMSType(client, MCMSTypeRegular)
}

// NewTimelockConfigurerWithMCMSType creates a TimelockConfigurer that targets
// either the standard MCMS or the CurseMCMS contract depending on mcmsType.
func NewTimelockConfigurerWithMCMSType(client aptos.AptosRpcClient, mcmsType MCMSType) *TimelockConfigurer {
	encoderFn := func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockUpdateDelayEncoder {
		return mcms.Bind(address, client).MCMS().Encoder()
	}
	if mcmsType == MCMSTypeCurse {
		encoderFn = func(address aptos.AccountAddress, client aptos.AptosRpcClient) timelockUpdateDelayEncoder {
			return curse_mcms.Bind(address, client).CurseMCMS().Encoder()
		}
	}

	return &TimelockConfigurer{
		client:    client,
		mcmsType:  mcmsType,
		encoderFn: encoderFn,
	}
}

// UpdateDelay encodes a TimelockUpdateMinDelay call on the Aptos MCMS module
// for the given address and returns it as a prepared MCMS transaction.
func (c *TimelockConfigurer) UpdateDelay(
	ctx context.Context, mcmsAddr string, newDelay uint64,
) (types.TransactionResult, error) {
	mcmsAddress, err := hexToAddress(mcmsAddr)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("failed to parse MCMS address %q: %w", mcmsAddr, err)
	}
	encoder := c.encoderFn(mcmsAddress, c.client)

	moduleInfo, function, _, args, err := encoder.TimelockUpdateMinDelay(newDelay)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("encoding TimelockUpdateMinDelay: %w", err)
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
		Hash:        "",
		ChainFamily: chainsel.FamilyAptos,
		RawData:     tx,
	}, nil
}

// GrantRole grants a timelock role to an address.
func (c *TimelockConfigurer) GrantRole(
	ctx context.Context,
	timelockAddress string,
	role sdk.TimelockRole,
	targetAddress string,
) (types.TransactionResult, error) {
	panic("not implemented")
}
