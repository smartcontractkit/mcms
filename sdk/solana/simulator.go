package solana

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.Simulator = &Simulator{}

type Simulator struct {
	executor     *Executor
	instructions []solana.Instruction
}

func NewSimulator(executor *Executor) *Simulator {
	simulator := &Simulator{instructions: []solana.Instruction{}}
	simulator.executor = executor.withSendAndConfirmFn(simulator.collectInstructions)

	return simulator
}

func (s *Simulator) SimulateSetRoot(
	ctx context.Context, _ string,
	metadata types.ChainMetadata, proof []common.Hash, root [32]byte,
	validUntil uint32, sortedSignatures []types.Signature,
) error {
	s.instructions = []solana.Instruction{}
	_, err := s.executor.SetRoot(ctx, metadata, proof, root, validUntil, sortedSignatures)
	if err != nil {
		return err
	}

	return s.simulate(ctx)
}

func (s *Simulator) SimulateOperation(
	ctx context.Context, metadata types.ChainMetadata, operation types.Operation,
) error {
	s.instructions = []solana.Instruction{}
	nonce := uint32(0)
	proof := []common.Hash{}
	_, err := s.executor.ExecuteOperation(ctx, metadata, nonce, proof, operation)
	if err != nil {
		return err
	}

	return s.simulate(ctx)
}

func (s *Simulator) collectInstructions(
	ctx context.Context,
	client *rpc.Client,
	auth solana.PrivateKey,
	instructionBuilder any,
	commitmentType rpc.CommitmentType,
) (string, *rpc.GetTransactionResult, error) {
	instruction, err := validateAndBuildSolanaInstruction(instructionBuilder)
	if err != nil {
		return "", nil, fmt.Errorf("unable to validate and build instruction: %w", err)
	}
	s.instructions = append(s.instructions, instruction)

	result := &rpc.GetTransactionResult{
		Slot:        1,
		BlockTime:   pointerTo(solana.UnixTimeSeconds(time.Now().Unix())),
		Transaction: &rpc.TransactionResultEnvelope{},
		Meta:        &rpc.TransactionMeta{},
		Version:     1,
	}

	return "<simulated-transaction>", result, nil
}

// sendAndConfirm contains the common logic for simulating instructions.
func (s *Simulator) simulate(ctx context.Context) error {
	result, err := solanaCommon.SimulateTransactionWithOpts(ctx, s.executor.client, s.instructions,
		s.executor.auth, rpc.SimulateTransactionOpts{Commitment: rpc.CommitmentConfirmed})
	if err != nil {
		return fmt.Errorf("unable to simulate instruction: %w", err)
	}
	if result.Value.Err != nil {
		return SimulateError{result.Value}
	}

	return nil
}

type SimulateError struct {
	result *rpc.SimulateTransactionResult
}

func (e SimulateError) Error() string {
	return fmt.Sprintf("%#v", e.result.Err)
}

func (e SimulateError) Logs() []string {
	return e.result.Logs
}

func pointerTo[T any](v T) *T {
	return &v
}
