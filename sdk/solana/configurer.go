package solana

import (
	"context"
	"fmt"
	"math"

	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/smartcontractkit/chainlink-ccip/chains/solana/contracts/tests/config"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/mcms"

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
func (c *Configurer) SetConfig(mcmAddressHex string, cfg *types.Config, clearRoot bool) (string, error) {
	ctx, cancel := context.WithCancel(context.Background()) // FIXME: add context as a method parameter?
	defer cancel()

	groupQuorums, groupParents, signerAddresses, signerGroups, err := evmsdk.ExtractSetConfigInputs(cfg)
	if err != nil {
		return "", fmt.Errorf("unable to extract set config inputs: %w", err)
	}

	if len(signerAddresses) > math.MaxUint8 {
		return "", fmt.Errorf("too many signatures (max %d)", math.MaxUint8)
	}

	// FIXME: global variables are bad, mmkay?
	config.TestChainID = uint64(c.chainSelector)
	config.McmProgram = solana.MustPublicKeyFromBase58(mcmAddressHex)
	mcm.SetProgramID(config.McmProgram) // see https://github.com/gagliardetto/solana-go/issues/254

	mcmAddress := solana.MustPublicKeyFromBase58(mcmAddressHex)
	configPDA := mcms.McmConfigAddress(mcmName)
	rootMetadataPDA := mcms.RootMetadataAddress(mcmName)
	expiringRootAndOpCountPDA := mcms.ExpiringRootAndOpCountAddress(mcmName)
	configSignersPDA := mcms.McmConfigSignersAddress(mcmName)

	err = initializeMcmProgram(ctx, c.client, c.auth, uint64(c.chainSelector), mcmAddress, mcmName,
		configPDA, rootMetadataPDA, expiringRootAndOpCountPDA)
	if err != nil {
		return "", fmt.Errorf("unable to initialize mcm program: %w", err)
	}

	err = c.preloadSigners(ctx, mcmName, solanaSignerAddresses(signerAddresses), configPDA, configSignersPDA)
	if err != nil {
		return "", fmt.Errorf("unable to preload signatures: %w", err)
	}

	setConfigInstruction := mcm.NewSetConfigInstruction(mcmName, signerGroups, groupQuorums, groupParents,
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
	initSignersInstruction := mcm.NewInitSignersInstruction(mcmName, uint8(len(signerAddresses)), configPDA,
		configSignersPDA, c.auth.PublicKey(), solana.SystemProgramID)
	_, err := sendAndConfirm(ctx, c.client, c.auth, initSignersInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to initialize signers: %w", err)
	}

	for i, chunkIndex := range chunkIndexes(len(signerAddresses), config.MaxAppendSignerBatchSize) {
		appendSignersInstructions := mcm.NewAppendSignersInstruction(mcmName,
			signerAddresses[chunkIndex[0]:chunkIndex[1]], configPDA, configSignersPDA, c.auth.PublicKey())
		_, err := sendAndConfirm(ctx, c.client, c.auth, appendSignersInstructions, rpc.CommitmentConfirmed)
		if err != nil {
			return fmt.Errorf("unable to append signers (%d): %w", i, err)
		}
	}

	finalizeSignersInstruction := mcm.NewFinalizeSignersInstruction(mcmName, configPDA, configSignersPDA,
		c.auth.PublicKey())
	_, err = sendAndConfirm(ctx, c.client, c.auth, finalizeSignersInstruction, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("unable to finalize signers: %w", err)
	}

	return nil
}

func solanaSignerAddresses(evmAddresses []evmCommon.Address) [][20]uint8{
	solanaAddresses := make([][20]uint8, len(evmAddresses))
	for i := range evmAddresses {
		solanaAddresses[i] = [20]uint8(evmAddresses[i])
	}
	return solanaAddresses
}
