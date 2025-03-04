package solana

import (
	"context"
	"fmt"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	"github.com/smartcontractkit/mcms/sdk"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

// Configurer configures the MCM contract for Solana chains.
type Configurer struct {
	instructionCollection
	chainSelector types.ChainSelector
	client        *rpc.Client
	auth          solana.PrivateKey
	skipSend      bool
	authority     solana.PublicKey
}

// NewConfigurer creates a new Configurer for Solana chains.
//
// options:
//
//	WithDoNotSendInstructionsOnChain: when selected, the Configurer instance will not
//		send the Solana instructions to the blockchain.
func NewConfigurer(
	client *rpc.Client, auth solana.PrivateKey, chainSelector types.ChainSelector, options ...configurerOption,
) *Configurer {
	configurer := &Configurer{
		client:        client,
		auth:          auth,
		chainSelector: chainSelector,
		skipSend:      false,
		authority:     auth.PublicKey(),
	}
	for _, opt := range options {
		opt(configurer)
	}

	return configurer
}

type configurerOption func(*Configurer)

func WithDoNotSendInstructionsOnChain() configurerOption {
	return func(c *Configurer) {
		c.skipSend = true
	}
}

func WithAuthority(newAuth solana.PublicKey) configurerOption {
	return func(c *Configurer) {
		c.authority = newAuth
	}
}

// SetConfig sets the configuration for the MCM contract on the Solana chain.
//
// The list of instructions needed to set the configuration is returned in the
// `RawData` field. And if the instructions were sent on chain (which they are
// unless the `WithDoNotSendInstructionsOnChain` option was selected in the
// constructor), the signature of the last instruction is returned in the
// `Hash` field.
func (c *Configurer) SetConfig(
	ctx context.Context, mcmAddress string, cfg *types.Config, clearRoot bool,
) (types.TransactionResult, error) {
	programID, pdaSeed, err := ParseContractAddress(mcmAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}

	groupQuorums, groupParents, signerAddresses, signerGroups, err := evmsdk.ExtractSetConfigInputs(cfg)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	if len(signerAddresses) > config.MaxNumSigners {
		return types.TransactionResult{}, fmt.Errorf("too many signers (max %d)", config.MaxNumSigners)
	}

	// FIXME: global variables are bad, mmkay?
	// see https://github.com/gagliardetto/solana-go/issues/254
	bindings.SetProgramID(programID)

	configPDA, err := FindConfigPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}
	rootMetadataPDA, err := FindRootMetadataPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}
	expiringRootAndOpCountPDA, err := FindExpiringRootAndOpCountPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}
	configSignersPDA, err := FindConfigSignersPDA(programID, pdaSeed)
	if err != nil {
		return types.TransactionResult{}, err
	}

	clear(c.instructions)
	defer clear(c.instructions)

	err = c.preloadSigners(pdaSeed, solanaSignerAddresses(signerAddresses), configPDA, configSignersPDA)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to preload signatures: %w", err)
	}

	err = c.addInstruction("setConfig", bindings.NewSetConfigInstruction(
		pdaSeed,
		signerGroups,
		groupQuorums,
		groupParents,
		clearRoot,
		configPDA,
		configSignersPDA,
		rootMetadataPDA,
		expiringRootAndOpCountPDA,
		c.authority,
		solana.SystemProgramID))
	if err != nil {
		return types.TransactionResult{}, err
	}

	var signature string
	if !c.skipSend {
		signature, err = c.sendInstructions(ctx, c.client, c.auth)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to set config: %w", err)
		}
	}

	return types.TransactionResult{
		Hash:        signature,
		ChainFamily: chain_selectors.FamilySolana,
		RawData:     c.solanaInstructions(),
	}, nil
}

func (c *Configurer) preloadSigners(
	mcmName [32]byte,
	signerAddresses [][20]uint8,
	configPDA solana.PublicKey,
	configSignersPDA solana.PublicKey,
) error {
	err := c.addInstruction("initSigners", bindings.NewInitSignersInstruction(mcmName, uint8(len(signerAddresses)), //nolint:gosec
		configPDA, configSignersPDA, c.authority, solana.SystemProgramID))
	if err != nil {
		return err
	}

	for i, chunkIndex := range chunkIndexes(len(signerAddresses), config.MaxAppendSignerBatchSize) {
		err = c.addInstruction(fmt.Sprintf("appendSigners%d", i), bindings.NewAppendSignersInstruction(mcmName,
			signerAddresses[chunkIndex[0]:chunkIndex[1]], configPDA, configSignersPDA, c.authority))
		if err != nil {
			return err
		}
	}

	err = c.addInstruction("finalizeSigners", bindings.NewFinalizeSignersInstruction(mcmName, configPDA,
		configSignersPDA, c.authority))
	if err != nil {
		return err
	}

	return nil
}

func solanaSignerAddresses(evmAddresses []evmCommon.Address) [][20]uint8 {
	solanaAddresses := make([][20]uint8, len(evmAddresses))
	for i := range evmAddresses {
		solanaAddresses[i] = [20]uint8(evmAddresses[i])
	}

	return solanaAddresses
}

type labeledInstruction struct {
	solana.Instruction
	label string
}

type instructionCollection struct {
	instructions []labeledInstruction
}

func (c *instructionCollection) solanaInstructions() []solana.Instruction {
	solanaInstructions := make([]solana.Instruction, len(c.instructions))
	for i, instruction := range c.instructions {
		solanaInstructions[i] = instruction.Instruction
	}

	return solanaInstructions
}

func (c *instructionCollection) addInstruction(label string, instructionBuilder any) error {
	instruction, err := validateAndBuildSolanaInstruction(instructionBuilder)
	if err != nil {
		return fmt.Errorf("unable to validate and build %s instruction: %w", label, err)
	}

	c.instructions = append(c.instructions, labeledInstruction{instruction, label})

	return nil
}

func (c *instructionCollection) sendInstructions(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
) (string, error) {
	if len(auth) == 0 {
		return "", nil
	}

	var signature string
	var err error
	for i, instruction := range c.instructions {
		signature, _, err = sendAndConfirmInstructions(ctx, client, auth,
			[]solana.Instruction{instruction}, rpc.CommitmentConfirmed)
		if err != nil {
			return "", fmt.Errorf("unable to send instruction %d - %s: %w", i, instruction.label, err)
		}
	}

	return signature, nil
}
