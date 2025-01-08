package solana

import (
	"context"
	"fmt"
	"math"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"

	"github.com/smartcontractkit/mcms/sdk"
	evmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Configurer = &Configurer{}

// Configurer configures the MCM contract for EVM chains.
type Configurer struct {
	chainSelector types.ChainSelector
	client        *rpc.Client
	auth          solana.PrivateKey
}

// NewConfigurer creates a new Configurer for EVM chains.
func NewConfigurer(client *rpc.Client, auth solana.PrivateKey, chainSelector types.ChainSelector) *Configurer {
	return &Configurer{
		client:        client,
		auth:          auth,
		chainSelector: chainSelector,
	}
}

// SetConfig sets the configuration for the MCM contract on the EVM chain.
func (c *Configurer) SetConfig(ctx context.Context, mcmAddress string, cfg *types.Config, clearRoot bool) (string, error) {
	programID, pdaSeed, err := ParseContractAddress(mcmAddress)
	if err != nil {
		return "", err
	}

	// FIXME: reuse ExtractSetConfigInputs from sdk.evm or duplicate it?
	groupQuorums, groupParents, signerAddresses, signerGroups, err := evmsdk.ExtractSetConfigInputs(cfg)
	if err != nil {
		return "", fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	if len(signerAddresses) > math.MaxUint8 {
		return "", fmt.Errorf("too many signers (max %d)", math.MaxUint8)
	}

	// FIXME: global variables are bad, mmkay?
	// see https://github.com/gagliardetto/solana-go/issues/254
	bindings.SetProgramID(programID)

	configPDA, err := FindConfigPDA(programID, pdaSeed)
	if err != nil {
		return "", err
	}
	rootMetadataPDA, err := FindRootMetadataPDA(programID, pdaSeed)
	if err != nil {
		return "", err
	}
	expiringRootAndOpCountPDA, err := FindExpiringRootAndOpCountPDA(programID, pdaSeed)
	if err != nil {
		return "", err
	}
	configSignersPDA, err := FindConfigSignersPDA(programID, pdaSeed)
	if err != nil {
		return "", err
	}

	err = initializeMcmProgram(ctx, c.client, c.auth, uint64(c.chainSelector), programID, pdaSeed,
		configPDA, rootMetadataPDA, expiringRootAndOpCountPDA)
	if err != nil {
		return "", fmt.Errorf("unable to initialize mcm program: %w", err)
	}

	err = c.preloadSigners(ctx, pdaSeed, solanaSignerAddresses(signerAddresses), configPDA, configSignersPDA)
	if err != nil {
		return "", fmt.Errorf("unable to preload signatures: %w", err)
	}

	setConfigInstruction := bindings.NewSetConfigInstruction(pdaSeed, signerGroups, groupQuorums, groupParents,
		clearRoot, configPDA, configSignersPDA, rootMetadataPDA, expiringRootAndOpCountPDA,
		c.auth.PublicKey(), solana.SystemProgramID)
	signature, err := sendAndConfirm(ctx, c.client, c.auth, setConfigInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("unable to set config: %w", err)
	}

	return signature, nil
}

func (c *Configurer) preloadSigners(
	ctx context.Context,
	mcmName [32]byte,
	signerAddresses [][20]uint8,
	configPDA solana.PublicKey,
	configSignersPDA solana.PublicKey,
) error {
	initSignersInstruction := bindings.NewInitSignersInstruction(mcmName, uint8(len(signerAddresses)), configPDA, //nolint:gosec
		configSignersPDA, c.auth.PublicKey(), solana.SystemProgramID)
	_, err := sendAndConfirm(ctx, c.client, c.auth, initSignersInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to initialize signers: %w", err)
	}

	for i, chunkIndex := range chunkIndexes(len(signerAddresses), config.MaxAppendSignerBatchSize) {
		appendSignersInstructions := bindings.NewAppendSignersInstruction(mcmName,
			signerAddresses[chunkIndex[0]:chunkIndex[1]], configPDA, configSignersPDA, c.auth.PublicKey())
		_, aerr := sendAndConfirm(ctx, c.client, c.auth, appendSignersInstructions, rpc.CommitmentConfirmed)
		if aerr != nil {
			return fmt.Errorf("unable to append signers (%d): %w", i, aerr)
		}
	}

	finalizeSignersInstruction := bindings.NewFinalizeSignersInstruction(mcmName, configPDA, configSignersPDA,
		c.auth.PublicKey())
	_, err = sendAndConfirm(ctx, c.client, c.auth, finalizeSignersInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to finalize signers: %w", err)
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
