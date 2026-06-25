package solana

import (
	"context"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
)

var _ sdk.TimelockConfigurer = (*TimelockConfigurer)(nil)

// TimelockConfigurer configures timelock parameters on Solana chains.
type TimelockConfigurer struct {
	*TimelockInspector
	client *rpc.Client
	auth   solana.PrivateKey
}

// NewTimelockConfigurer creates a new TimelockConfigurer for Solana chains.
func NewTimelockConfigurer(client *rpc.Client, auth solana.PrivateKey) *TimelockConfigurer {
	return &TimelockConfigurer{
		TimelockInspector: NewTimelockInspector(client),
		client:            client,
		auth:              auth,
	}
}

// UpdateDelay calls the UpdateDelay instruction on the Solana RBACTimelock program.
func (c *TimelockConfigurer) UpdateDelay(
	ctx context.Context, timelockAddress string, newDelay uint64,
) (types.TransactionResult, error) {
	programID, timelockID, err := ParseContractAddress(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to parse contract address: %w", err)
	}

	bindings.SetProgramID(programID)

	configPDA, err := FindTimelockConfigPDA(programID, timelockID)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to find timelock config pda: %w", err)
	}

	instruction := bindings.NewUpdateDelayInstruction(
		timelockID,
		newDelay,
		configPDA,
		c.auth.PublicKey(),
	)

	signature, tx, err := sendAndConfirm(ctx, c.client, c.auth, instruction, rpc.CommitmentConfirmed)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to update delay: %w", err)
	}

	return types.TransactionResult{
		Hash:        signature,
		ChainFamily: chainsel.FamilySolana,
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
