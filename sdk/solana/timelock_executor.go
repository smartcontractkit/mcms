package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"
	solanaCommon "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/eth"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

var _ sdk.TimelockExecutor = (*TimelockExecutor)(nil)

// TimelockExecutor is an Executor implementation for solana chains for accessing the RBACTimelock program
type TimelockExecutor struct {
	TimelockInspector
	client *rpc.Client
	auth   solana.PrivateKey
}

// NewTimelockExecutor creates a new TimelockExecutor
func NewTimelockExecutor(client *rpc.Client, auth solana.PrivateKey) *TimelockExecutor {
	return &TimelockExecutor{
		TimelockInspector: *NewTimelockInspector(client),
		client:            client,
		auth:              auth,
	}
}

// Execute runs ExecuteBatch ix for each transaction in the BatchOperation
func (t *TimelockExecutor) Execute(ctx context.Context, bop types.BatchOperation, timelockAddress string, predecessor common.Hash, salt common.Hash) (string, error) {
	programID, timelockID, err := ParseContractAddress(timelockAddress)
	if err != nil {
		return "", err
	}
	timelock.SetProgramID(programID) // see https://github.com/gagliardetto/solana-go/issues/254

	ixs := make([]timelock.InstructionData, len(bop.Transactions))
	var additionalFields AdditionalFields
	for i, tx := range bop.Transactions {
		// Unmarshal the AdditionalFields from the operation

		if err = json.Unmarshal(tx.AdditionalFields, &additionalFields); err != nil {
			return "", err
		}
		accounts := make([]timelock.InstructionAccount, len(additionalFields.Accounts))
		for i, acc := range additionalFields.Accounts {
			accounts[i] = timelock.InstructionAccount{
				Pubkey:     acc.PublicKey,
				IsSigner:   acc.IsSigner,
				IsWritable: acc.IsWritable,
			}
		}
		var toProgramID solana.PublicKey
		toProgramID, err = solana.PublicKeyFromBase58(tx.To)
		if err != nil {
			return "", fmt.Errorf("unable to get hash from base58 To address: %w", err)
		}
		ixs[i] = timelock.InstructionData{
			Data:      tx.Data,
			Accounts:  accounts,
			ProgramId: toProgramID,
		}
	}
	var predBytes [32]byte
	copy(predBytes[:], predecessor.Bytes())

	operationID := HashOperation(ixs, predBytes, salt)
	operationPDA, err := FindTimelockOperationPDA(programID, timelockID, operationID)
	if err != nil {
		return "", err
	}

	configPDA, err := FindTimelockConfigPDA(programID, timelockID)
	if err != nil {
		return "", err
	}
	signerPDA, err := FindTimelockSignerPDA(programID, timelockID)
	if err != nil {
		return "", err
	}
	var configAccount timelock.Config
	err = solanaCommon.GetAccountDataBorshInto(ctx, t.client, configPDA, rpc.CommitmentConfirmed, &configAccount)
	if err != nil {
		return "", err
	}
	controller, err := getRoleAccessController(configAccount, timelock.Executor_Role)
	if err != nil {
		return "", err
	}

	ix := timelock.NewExecuteBatchInstruction(
		timelockID,
		operationID,
		operationPDA,
		predBytes,
		configPDA,
		signerPDA,
		controller,
		t.auth.PublicKey())

	// Add accounts from the operation to execute
	ix.AccountMetaSlice = append(ix.AccountMetaSlice, additionalFields.Accounts...)

	builtIx, err := ix.ValidateAndBuild()
	if err != nil {
		return "", fmt.Errorf("unable to validate and build instruction: %w", err)
	}

	signature, err := sendAndConfirmBuiltIx(ctx, t.client, t.auth, builtIx, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("unable to call execute operation instruction: %w", err)
	}

	return signature, nil
}

// HashOperation hashes the operation and returns the operation ID
func HashOperation(instructions []timelock.InstructionData, predecessor [32]byte, salt [32]byte) [32]byte {
	var encodedData bytes.Buffer

	for _, ix := range instructions {
		encodedData.Write(ix.ProgramId[:])

		for _, acc := range ix.Accounts {
			encodedData.Write(acc.Pubkey[:])
			if acc.IsSigner {
				encodedData.WriteByte(1)
			} else {
				encodedData.WriteByte(0)
			}
			if acc.IsWritable {
				encodedData.WriteByte(1)
			} else {
				encodedData.WriteByte(0)
			}
		}
		encodedData.Write(ix.Data)
	}

	encodedData.Write(predecessor[:])
	encodedData.Write(salt[:])

	result := eth.Keccak256(encodedData.Bytes())

	var hash [32]byte
	copy(hash[:], result)

	return hash
}
