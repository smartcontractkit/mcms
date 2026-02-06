package solana

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_1/timelock"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for solana chains for accessing the RBACTimelock program
type TimelockExecutor struct {
	*TimelockInspector
	client *rpc.Client
	auth   solana.PrivateKey
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client *rpc.Client, auth solana.PrivateKey) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: NewTimelockInspector(client),
		client:            client,
		auth:              auth,
	}
}

func (e *TimelockExecutor) Client() *rpc.Client {
	return e.client
}

// Execute runs the ExecuteBatch instruction for each transaction in the BatchOperation
func (e *TimelockExecutor) Execute(
	ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash,
) (types.TransactionResult, error) {
	programID, timelockID, err := ParseContractAddress(timelockAddress)
	if err != nil {
		return types.TransactionResult{}, err
	}
	bindings.SetProgramID(programID) // see https://github.com/gagliardetto/solana-go/issues/254

	instructionsData, err := getInstructionDataFromBatchOperation(bop)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to get InstructionData from batch operation: %w", err)
	}
	accounts, err := getAccountsFromBatchOperation(bop)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to get accounts from batch operation: %w", err)
	}

	operationID, err := HashOperation(instructionsData, predecessor, salt)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to compute operation id: %w", err)
	}

	operationPDA, err := FindTimelockOperationPDA(programID, timelockID, operationID)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to find timelock operation pda: %w", err)
	}
	configPDA, err := FindTimelockConfigPDA(programID, timelockID)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to find timelock config pda: %w", err)
	}
	signerPDA, err := FindTimelockSignerPDA(programID, timelockID)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to find timelock signer pda: %w", err)
	}
	config, err := GetTimelockConfig(ctx, e.client, configPDA)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to read config pda: %w", err)
	}

	var predecessorOperationPDA solana.PublicKey
	if (predecessor != common.Hash{}) {
		predecessorOperationPDA, err = FindTimelockOperationPDA(programID, timelockID, predecessor)
		if err != nil {
			return types.TransactionResult{}, fmt.Errorf("unable to find timelock predecessor operation pda: %w", err)
		}
	}

	instruction := bindings.NewExecuteBatchInstruction(timelockID, operationID, operationPDA,
		predecessorOperationPDA, configPDA, signerPDA, config.ExecutorRoleAccessController, e.auth.PublicKey())
	instruction.AccountMetaSlice = append(instruction.AccountMetaSlice, accounts...)

	signature, tx, err := sendAndConfirm(ctx, e.client, e.auth, instruction, rpc.CommitmentConfirmed)
	if err != nil {
		return types.TransactionResult{}, fmt.Errorf("unable to call execute operation instruction: %w", err)
	}

	return types.TransactionResult{
		Hash:        signature,
		ChainFamily: chainsel.FamilySolana,
		RawData:     tx,
	}, nil
}
