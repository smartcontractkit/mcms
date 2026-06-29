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
	instructionCollection
	client           *rpc.Client
	auth             solana.PrivateKey
	skipSend         bool
	authorityAccount solana.PublicKey
}

type timelockConfigurerOption func(*TimelockConfigurer)

// WithDoNotSendTimelockInstructionsOnChain configures the TimelockConfigurer to build
// transactions without sending them on chain.
func WithDoNotSendTimelockInstructionsOnChain() timelockConfigurerOption {
	return func(c *TimelockConfigurer) {
		c.skipSend = true
	}
}

// WithTimelockAuthorityAccount sets the authority account for timelock instructions.
// Defaults to the auth public key when unset.
func WithTimelockAuthorityAccount(authorityAccount solana.PublicKey) timelockConfigurerOption {
	return func(c *TimelockConfigurer) {
		c.authorityAccount = authorityAccount
	}
}

// NewTimelockConfigurer creates a new TimelockConfigurer for Solana chains.
func NewTimelockConfigurer(client *rpc.Client, auth solana.PrivateKey, options ...timelockConfigurerOption) *TimelockConfigurer {
	configurer := &TimelockConfigurer{
		TimelockInspector: NewTimelockInspector(client),
		client:            client,
		auth:              auth,
		authorityAccount:  auth.PublicKey(),
	}
	for _, opt := range options {
		opt(configurer)
	}

	return configurer
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

// GrantRole grants a timelock role to an address via the BatchAddAccess instruction.
func (c *TimelockConfigurer) GrantRole(
	ctx context.Context,
	timelockAddress string,
	role sdk.TimelockRole,
	targetAddress string,
) (types.TransactionResult, error) {
	target, err := solana.PublicKeyFromBase58(targetAddress)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("invalid target address: %s", targetAddress)
	}
	if target.IsZero() {
		return types.TransactionResult{}, fmt.Errorf("invalid target address: %s", targetAddress)
	}

	instructionBuilder, err := newGrantRoleInstructionBuilder(
		ctx, c.client, timelockAddress, role, target, c.authorityAccount,
	)
	if err != nil {
		return types.TransactionResult{}, err
	}

	defer func() { c.instructions = []labeledInstruction{} }()

	err = c.addInstruction("GrantRole", instructionBuilder)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to build grant role instruction: %w", err)
	}

	var signature string
	if !c.skipSend {
		signature, err = c.sendInstructions(ctx, c.client, c.auth)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to grant role: %w", err)
		}
	}

	var rawData any
	if c.skipSend {
		rawData, err = NewTransactionFromInstruction(
			c.instructions[0].Instruction,
			rbacTimelockContractType,
			[]string{rbacTimelockContractType, "GrantRole"},
		)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to build grant role transaction: %w", err)
		}
	} else {
		rawData = c.solanaInstructions()
	}

	return types.TransactionResult{
		Hash:        signature,
		ChainFamily: chainsel.FamilySolana,
		RawData:     rawData,
	}, nil
}

func newGrantRoleInstructionBuilder(
	ctx context.Context,
	client *rpc.Client,
	timelockAddress string,
	role sdk.TimelockRole,
	target solana.PublicKey,
	authority solana.PublicKey,
) (timelockInstructionBuilder, error) {
	programID, timelockID, err := ParseContractAddress(timelockAddress)
	if err != nil {
		return nil, fmt.Errorf("unable to parse contract address: %w", err)
	}

	bindings.SetProgramID(programID)

	bindingRole, err := TimelockRoleToBinding(role)
	if err != nil {
		return nil, err
	}

	configPDA, err := FindTimelockConfigPDA(programID, timelockID)
	if err != nil {
		return nil, fmt.Errorf("unable to find timelock config pda: %w", err)
	}

	config, err := GetTimelockConfig(ctx, client, configPDA)
	if err != nil {
		return nil, fmt.Errorf("unable to read timelock config: %w", err)
	}

	roleAccessController, err := getRoleAccessController(config, bindingRole)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve role access controller: %w", err)
	}

	accessControllerProgramID, err := getAccountOwner(ctx, client, roleAccessController)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve access controller program id: %w", err)
	}

	instructionBuilder := bindings.NewBatchAddAccessInstruction(
		timelockID,
		bindingRole,
		configPDA,
		accessControllerProgramID,
		roleAccessController,
		authority,
	)
	instructionBuilder.Append(solana.Meta(target))

	return instructionBuilder, nil
}

func getAccountOwner(ctx context.Context, client *rpc.Client, account solana.PublicKey) (solana.PublicKey, error) {
	info, err := client.GetAccountInfoWithOpts(ctx, account, &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("unable to get account info for %s: %w", account, err)
	}
	if info == nil || info.Value == nil {
		return solana.PublicKey{}, fmt.Errorf("account not found: %s", account)
	}

	return info.Value.Owner, nil
}
