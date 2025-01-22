package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/mcm"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
)

type Simulatable struct {
	simulate     bool
	instructions []instructionBuilder[*mcm.Instruction]
}

func (s *Simulatable) EnableSimulation(
	ctx context.Context, client *rpc.Client, auth solana.PrivateKey, opts rpc.SimulateTransactionOpts, action func() error,
) error {
	s.simulate = true

	err := action()
	if err != nil {
		return err
	}

	err = simulateInstructionWithOpts(ctx, client, auth, s.instructions, opts)
	s.simulate = false
	s.instructions = nil
	if err != nil {
		return fmt.Errorf("simulation failed: %w", err)
	}

	return nil
}

func (s *Simulatable) sendAndConfirmOrSimulate(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructionBuilder instructionBuilder[*mcm.Instruction],
	commitmentType rpc.CommitmentType,
) (string, error) {
	if s.simulate {
		s.instructions = append(s.instructions, instructionBuilder)
		return "", nil
	}

	signature, err := sendAndConfirm(ctx, client, auth, instructionBuilder, commitmentType)

	return signature, err
}

func simulateInstructionWithOpts(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructionBuilders []instructionBuilder[*mcm.Instruction],
	opts rpc.SimulateTransactionOpts,
) error {
	insts := make([]solana.Instruction, 0, len(instructionBuilders))
	for _, inst := range instructionBuilders {
		builtInstruction, err := inst.ValidateAndBuild()
		if err != nil {
			return fmt.Errorf("unable to validate and build instruction: %w", err)
		}
		insts = append(insts, builtInstruction)
	}

	result, err := solanaCommon.SimulateTransactionWithOpts(ctx, client, insts, auth, opts)
	if err != nil {
		return fmt.Errorf("unable to simulate instruction: %w", err)
	}
	if result.Value.Err != nil {
		return fmt.Errorf("unable to simulate instruction: %s", result.Value.Err)
	}

	return nil
}
