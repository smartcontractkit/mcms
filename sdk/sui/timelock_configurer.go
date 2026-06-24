package sui

import (
	"context"
	"fmt"

	"github.com/aptos-labs/aptos-go-sdk/bcs"
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

const suiTimelockUpdateMinDelayModuleName = "mcms"

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

// TimelockConfigurer configures timelock parameters on Sui chains.
// UpdateDelay returns a prepared MCMS transaction instead of executing on-chain.
type TimelockConfigurer struct {
	mcmsPackageID string
}

// NewTimelockConfigurer creates a new TimelockConfigurer for Sui chains.
func NewTimelockConfigurer(mcmsPackageID string) *TimelockConfigurer {
	return &TimelockConfigurer{mcmsPackageID: mcmsPackageID}
}

// UpdateDelay prepares the Sui MCMS transaction for a timelock min-delay
// update and returns it as a prepared MCMS transaction.
func (c *TimelockConfigurer) UpdateDelay(
	ctx context.Context, timelockAddress string, newDelay uint64,
) (types.TransactionResult, error) {
	if timelockAddress == "" {
		return types.TransactionResult{}, fmt.Errorf("timelock address is required")
	}

	data, err := serializeTimelockUpdateMinDelay(timelockAddress, newDelay)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("encoding timelock_update_min_delay: %w", err)
	}

	// chainlink-sui does not generate the plain timelock_update_min_delay.
	tx, err := NewTransactionWithStateObj(
		suiTimelockUpdateMinDelayModuleName,
		suiTimelockUpdateMinDelayFunctionName,
		c.mcmsPackageID,
		data,
		"",
		nil,
		timelockAddress,
		nil,
	)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("creating mcms transaction: %w", err)
	}

	return types.TransactionResult{
		Hash:        "",
		ChainFamily: chainsel.FamilySui,
		RawData:     tx,
	}, nil
}

// serializeTimelockUpdateMinDelay BCS-encodes the timelock object address and new delay.
func serializeTimelockUpdateMinDelay(timelockAddress string, newMinDelay uint64) ([]byte, error) {
	timelockAddr, err := AddressFromHex(timelockAddress)
	if err != nil {
		return nil, fmt.Errorf("decoding timelock address: %w", err)
	}

	return bcs.SerializeSingle(func(ser *bcs.Serializer) {
		ser.FixedBytes(timelockAddr.Bytes())
		ser.U64(newMinDelay)
	})
}

// GrantRole grants a timelock role to an address.
func (c *TimelockConfigurer) GrantRole(
	ctx context.Context,
	timelockAddress string,
	role sdk.TimelockRole,
	address string,
) (types.TransactionResult, error) {
	panic("not implemented")
}
