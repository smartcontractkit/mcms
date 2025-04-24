package solana

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	bindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/timelock"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// AppendIxDataChunkSize number is derived from chainlink-ccip
// https://github.com/smartcontractkit/chainlink-ccip/blob/main/chains/solana/contracts/tests/config/timelock_config.go#L20
const AppendIxDataChunkSize = 491

var _ sdk.TimelockConverter = (*TimelockConverter)(nil)

type TimelockConverter struct {
}

func (t TimelockConverter) ConvertBatchToChainOperations(
	ctx context.Context,
	metadata types.ChainMetadata,
	batchOp types.BatchOperation,
	timelockAddress string,
	mcmAddress string,
	delay types.Duration,
	action types.TimelockAction,
	predecessor common.Hash,
	salt common.Hash,
) ([]types.Operation, common.Hash, error) {
	timelockProgramID, timelockPDASeed, err := ParseContractAddress(timelockAddress)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to parse timelock address: %w", err)
	}
	mcmProgramID, mcmPDASeed, err := ParseContractAddress(mcmAddress)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to parse mcm address: %w", err)
	}

	bindings.SetProgramID(timelockProgramID)

	tags := getTagsFromBatchOperation(batchOp)
	instructionsData, err := getInstructionDataFromBatchOperation(batchOp, action == types.TimelockActionBypass)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to convert batchop to solana instructions: %w", err)
	}

	operationID, err := HashOperation(instructionsData, predecessor, salt)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to compute operation id: %w", err)
	}

	operationPDA, err := FindTimelockOperationPDA(timelockProgramID, timelockPDASeed, operationID)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to find timelock operation pda: %w", err)
	}
	operationBypasserPDA, err := FindTimelockBypasserOperationPDA(timelockProgramID, timelockPDASeed, operationID)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to find timelock bypasser operation pda: %w", err)
	}
	configPDA, err := FindTimelockConfigPDA(timelockProgramID, timelockPDASeed)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to find timelock config pda: %w", err)
	}
	signerPDA, err := FindTimelockSignerPDA(timelockProgramID, timelockPDASeed)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to find timelock signer pda: %w", err)
	}
	mcmSignerPDA, err := FindSignerPDA(mcmProgramID, mcmPDASeed)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to find mcm signer address: %w", err)
	}
	var additionalFields AdditionalFieldsMetadata
	if err = json.Unmarshal(metadata.AdditionalFields, &additionalFields); err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to unmarshal solana-specific additional fields from chain metada: %w", err)
	}
	// encode the data based on the operation
	var instructions []solana.Instruction
	switch action {
	case types.TimelockActionSchedule:
		instructions, err = scheduleBatchInstructions(timelockPDASeed, operationID, predecessor, salt, delay.Duration,
			uint32(len(batchOp.Transactions)), instructionsData, additionalFields.ProposerRoleAccessController, operationPDA, //nolint:gosec
			configPDA, mcmSignerPDA)
	case types.TimelockActionCancel:
		instructions, err = cancelInstructions(timelockPDASeed, operationID, additionalFields.CancellerRoleAccessController,
			operationPDA, configPDA, mcmSignerPDA)
	case types.TimelockActionBypass:
		instructions, err = bypassInstructions(timelockPDASeed, operationID, additionalFields.BypasserRoleAccessController,
			operationBypasserPDA, configPDA, signerPDA, mcmSignerPDA, salt, uint32(len(batchOp.Transactions)), instructionsData) //nolint:gosec
	default:
		err = fmt.Errorf("invalid timelock operation: %s", string(action))
	}
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to build %v instruction: %w", action, err)
	}

	operations, err := solanaInstructionToMcmsOperation(instructions, batchOp.ChainSelector, tags)
	if err != nil {
		return []types.Operation{}, common.Hash{}, fmt.Errorf("unable to convert instructions to mcms operations: %w", err)
	}

	return operations, operationID, nil
}

// HashOperation replicates the hash calculation from Solidity
func HashOperation(instructions []bindings.InstructionData, predecessor, salt [32]byte) (common.Hash, error) {
	var encoded bytes.Buffer

	err := binary.Write(&encoded, binary.LittleEndian, uint32(len(instructions))) //nolint:gosec
	if err != nil {
		return [32]byte{}, fmt.Errorf("unable to write number of instructions: %w", err)
	}

	for _, ix := range instructions {
		encoded.Write(ix.ProgramId[:])

		err := binary.Write(&encoded, binary.LittleEndian, uint32(len(ix.Accounts))) //nolint:gosec
		if err != nil {
			return [32]byte{}, fmt.Errorf("unable to write number of accounts: %w", err)
		}

		for _, acc := range ix.Accounts {
			encoded.Write(acc.Pubkey[:])
			encoded.WriteByte(boolToByte(acc.IsSigner))
			encoded.WriteByte(boolToByte(acc.IsWritable))
		}

		err = binary.Write(&encoded, binary.LittleEndian, uint32(len(ix.Data))) //nolint:gosec
		if err != nil {
			return [32]byte{}, fmt.Errorf("unable to write data size: %w", err)
		}
		encoded.Write(ix.Data)
	}

	encoded.Write(predecessor[:])
	encoded.Write(salt[:])

	return crypto.Keccak256Hash(encoded.Bytes()), nil
}

func accountMetaToInstructionAccount(accounts ...*solana.AccountMeta) []bindings.InstructionAccount {
	instructionAccounts := make([]bindings.InstructionAccount, len(accounts))
	for i, account := range accounts {
		instructionAccounts[i] = bindings.InstructionAccount{
			Pubkey:     account.PublicKey,
			IsSigner:   account.IsSigner,
			IsWritable: account.IsWritable,
		}
	}

	return instructionAccounts
}

func getInstructionDataFromBatchOperation(
	batchOp types.BatchOperation, addProgramIDToAccounts bool,
) ([]bindings.InstructionData, error) {
	instructionsData := make([]bindings.InstructionData, 0)
	for _, tx := range batchOp.Transactions {
		toProgramID, err := ParseProgramID(tx.To)
		if err != nil {
			return nil, fmt.Errorf("unable to parse program id from To field: %w", err)
		}

		var additionalFields AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			err = json.Unmarshal(tx.AdditionalFields, &additionalFields)
			if err != nil {
				return nil, fmt.Errorf("unable to unmarshal additional fields: %w\n%v", err, string(tx.AdditionalFields))
			}
		}

		if addProgramIDToAccounts {
			additionalFields.Accounts = append(additionalFields.Accounts, &solana.AccountMeta{PublicKey: toProgramID})
		}

		instructionsData = append(instructionsData, bindings.InstructionData{
			ProgramId: toProgramID,
			Data:      tx.Data,
			Accounts:  accountMetaToInstructionAccount(additionalFields.Accounts...),
		})
	}

	return instructionsData, nil
}

func getAccountsFromBatchOperation(batchOp types.BatchOperation) ([]*solana.AccountMeta, error) {
	accounts := make([]*solana.AccountMeta, 0)
	for _, tx := range batchOp.Transactions {
		toProgramID, err := ParseProgramID(tx.To)
		if err != nil {
			return nil, fmt.Errorf("unable to parse program id from To field: %w", err)
		}
		accounts = append(accounts, &solana.AccountMeta{PublicKey: toProgramID})

		var additionalFields AdditionalFields
		if len(tx.AdditionalFields) > 0 {
			err = json.Unmarshal(tx.AdditionalFields, &additionalFields)
			if err != nil {
				return nil, fmt.Errorf("unable to unmarshal additional fields: %w\n%v", err, string(tx.AdditionalFields))
			}
		}
		accounts = append(accounts, additionalFields.Accounts...)
	}

	return accounts, nil
}

func getTagsFromBatchOperation(batchOp types.BatchOperation) []string {
	tags := make([]string, 0)
	for _, tx := range batchOp.Transactions {
		tags = append(tags, tx.Tags...)
	}

	return tags
}

func solanaInstructionToMcmsOperation(
	instructions []solana.Instruction, chainSelector types.ChainSelector, tags []string,
) ([]types.Operation, error) {
	operations := make([]types.Operation, 0, len(instructions))
	for _, instruction := range instructions {
		data, err := instruction.Data()
		if err != nil {
			return []types.Operation{}, fmt.Errorf("unable to get instruction data: %w", err)
		}

		accounts := []*solana.AccountMeta{}
		for _, account := range instruction.Accounts() {
			account.IsSigner = account.IsSigner && solana.IsOnCurve(account.PublicKey.Bytes())
			accounts = append(accounts, account)
		}

		transaction, err := NewTransaction(instruction.ProgramID().String(), data, (*big.Int)(nil),
			accounts, "RBACTimelock", tags)
		if err != nil {
			return []types.Operation{}, fmt.Errorf("unable to create new transaction: %w", err)
		}

		operations = append(operations, types.Operation{ChainSelector: chainSelector, Transaction: transaction})
	}

	return operations, nil
}

func scheduleBatchInstructions(
	pdaSeed PDASeed, operationID, predecessor, salt [32]byte, delay time.Duration,
	numInstructions uint32, instructionsData []bindings.InstructionData,
	proposerAccessController, operationPDA, configPDA, mcmSignerPDA solana.PublicKey,
) ([]solana.Instruction, error) {
	instructions := make([]solana.Instruction, 0, numInstructions)

	// initialize
	instruction, err := bindings.NewInitializeOperationInstruction(pdaSeed, operationID, predecessor, salt,
		numInstructions, operationPDA, configPDA,
		proposerAccessController,
		mcmSignerPDA, solana.SystemProgramID).ValidateAndBuild()
	if err != nil {
		return []solana.Instruction{}, fmt.Errorf("unable to build InitializeOperation instruction: %w", err)
	}
	instructions = append(instructions, instruction)

	for i, ixData := range instructionsData {
		initIx, initializeErr := bindings.NewInitializeInstructionInstruction(
			pdaSeed, operationID,
			ixData.ProgramId,
			ixData.Accounts,
			operationPDA,
			configPDA,
			proposerAccessController,
			mcmSignerPDA,
			solana.SystemProgramID,
		).ValidateAndBuild()
		if initializeErr != nil {
			return []solana.Instruction{}, fmt.Errorf("unable to build InitializeInstruction instruction (ixIndex=%d): %w", i, initializeErr)
		}
		instructions = append(instructions, initIx)

		rawData := ixData.Data
		offset := 0

		for offset < len(rawData) {
			end := offset + AppendIxDataChunkSize
			if end > len(rawData) {
				end = len(rawData)
			}
			chunk := rawData[offset:end]

			appendIx, appendErr := bindings.NewAppendInstructionDataInstruction(
				pdaSeed,
				operationID,
				//nolint:gosec
				uint32(i), // which instruction index we are chunking
				chunk,     // partial data
				operationPDA,
				configPDA,
				proposerAccessController,
				mcmSignerPDA,
				solana.SystemProgramID,
			).ValidateAndBuild()
			if appendErr != nil {
				return nil, fmt.Errorf("unable to build AppendInstructionData instruction (ixIndex=%d): %w", i, appendErr)
			}
			instructions = append(instructions, appendIx)

			offset = end
		}
	}

	instruction, err = bindings.NewFinalizeOperationInstruction(pdaSeed, operationID,
		operationPDA, configPDA, proposerAccessController, mcmSignerPDA).ValidateAndBuild()
	if err != nil {
		return []solana.Instruction{}, fmt.Errorf("unable to build FinializeOperation instruction: %w", err)
	}
	instructions = append(instructions, instruction)

	// schedule batch
	instruction, err = bindings.NewScheduleBatchInstruction(pdaSeed, operationID, uint64(delay.Seconds()),
		operationPDA, configPDA, proposerAccessController, mcmSignerPDA).ValidateAndBuild()
	if err != nil {
		return []solana.Instruction{}, fmt.Errorf("unable to build ScheduleBatch instruction: %w", err)
	}
	instructions = append(instructions, instruction)

	return instructions, nil
}

func cancelInstructions(
	pdaSeed PDASeed, operationID [32]byte, cancelAccessController, operationPDA, configPDA, mcmSignerPDA solana.PublicKey,
) ([]solana.Instruction, error) {
	instruction, err := bindings.NewCancelInstruction(pdaSeed, operationID, operationPDA, configPDA,
		cancelAccessController, mcmSignerPDA).ValidateAndBuild()
	if err != nil {
		return []solana.Instruction{}, fmt.Errorf("unable to build Cancel instruction: %w", err)
	}

	return []solana.Instruction{instruction}, nil
}

func bypassInstructions(
	pdaSeed PDASeed, operationID [32]byte, bypassAccessController, operationPDA, configPDA, signerPDA,
	mcmSignerPDA solana.PublicKey,
	salt [32]byte,
	numInstructions uint32, instructionsData []bindings.InstructionData,
) ([]solana.Instruction, error) {
	instructions := make([]solana.Instruction, 0, numInstructions)

	// -- initialize bypasser operation
	initOpIx, ioErr := bindings.NewInitializeBypasserOperationInstruction(
		pdaSeed,
		operationID,
		salt,
		numInstructions,
		operationPDA,
		configPDA,
		bypassAccessController,
		mcmSignerPDA,
		solana.SystemProgramID,
	).ValidateAndBuild()
	if ioErr != nil {
		return nil, fmt.Errorf("unable to build InitializeBypasserOperation instruction: %w", ioErr)
	}
	instructions = append(instructions, initOpIx)

	embeddedInstructionsAccounts := []*solana.AccountMeta{}
	for i, instruction := range instructionsData {
		for _, acc := range instruction.Accounts {
			embeddedInstructionsAccounts = append(embeddedInstructionsAccounts, &solana.AccountMeta{
				PublicKey: acc.Pubkey, IsWritable: acc.IsWritable, IsSigner: acc.IsSigner,
			})
		}

		// -- initialize bypasser instruction
		initIx, apErr := bindings.NewInitializeBypasserInstructionInstruction(
			pdaSeed,
			operationID,
			instruction.ProgramId, // ProgramId
			instruction.Accounts,  //
			operationPDA,
			configPDA,
			bypassAccessController,
			mcmSignerPDA,
			solana.SystemProgramID, // for reallocation
		).ValidateAndBuild()
		if apErr != nil {
			return nil, fmt.Errorf("unable to build InitializeBypasserInstruction instruction (ixIndex=%d): %w", i, apErr)
		}
		instructions = append(instructions, initIx)

		rawData := instruction.Data
		offset := 0

		for offset < len(rawData) {
			// -- append bypasser instruction data
			end := offset + AppendIxDataChunkSize
			if end > len(rawData) {
				end = len(rawData)
			}
			chunk := rawData[offset:end]

			appendIx, appendErr := bindings.NewAppendBypasserInstructionDataInstruction(
				pdaSeed,
				operationID,
				//nolint:gosec
				uint32(i), // which instruction index we are chunking
				chunk,     // partial data
				operationPDA,
				configPDA,
				bypassAccessController,
				mcmSignerPDA,
				solana.SystemProgramID, // for reallocation
			).ValidateAndBuild()
			if appendErr != nil {
				return nil, fmt.Errorf("unable to build AppendBypasserInstruction instruction (ixIndex=%d): %w", i, appendErr)
			}
			instructions = append(instructions, appendIx)

			offset = end
		}
	}

	// -- finalize bypasser operation
	finOpIx, foErr := bindings.NewFinalizeBypasserOperationInstruction(
		pdaSeed,
		operationID,
		operationPDA,
		configPDA,
		bypassAccessController,
		mcmSignerPDA,
	).ValidateAndBuild()
	if foErr != nil {
		return nil, fmt.Errorf("failed to build finalize bypasser operation instruction: %w", foErr)
	}
	instructions = append(instructions, finOpIx)

	// -- bypasser execute batch
	instructionBuilder := bindings.NewBypasserExecuteBatchInstruction(pdaSeed, operationID, operationPDA,
		configPDA, signerPDA, bypassAccessController, mcmSignerPDA)
	// instructionBuilder.GetTimelockSignerAccount().IsWritable = true
	// TODO: how do we resolve conflicts when IsSigner or IsWritable differ for the same account
	accountsMap := map[solana.PublicKey]*solana.AccountMeta{}
	for _, accountMeta := range instructionBuilder.AccountMetaSlice {
		accountsMap[accountMeta.PublicKey] = accountMeta
	}

	for _, account := range embeddedInstructionsAccounts {
		existingAccount, found := accountsMap[account.PublicKey]
		if found {
			existingAccount.IsSigner = existingAccount.IsSigner || account.IsSigner
			existingAccount.IsWritable = existingAccount.IsWritable || account.IsWritable
			instructionBuilder.AccountMetaSlice = append(instructionBuilder.AccountMetaSlice, account)
		} else {
			accountsMap[account.PublicKey] = account
			instructionBuilder.AccountMetaSlice = append(instructionBuilder.AccountMetaSlice, account)
		}
	}

	// fmt.Printf("*** FINAL ACCOUNTS FOR BYPASS EXECUTION ***\n")
	// for _, acc := range instructionBuilder.AccountMetaSlice {
	// 	fmt.Printf("    %s [signer: %v] [writable: %v]\n", acc.PublicKey, acc.IsSigner, acc.IsWritable)
	// }

	instruction, err := instructionBuilder.ValidateAndBuild()
	if err != nil {
		return []solana.Instruction{}, fmt.Errorf("unable to build BypasserExecuteBatch instruction: %w", err)
	}
	instructions = append(instructions, instruction)

	return instructions, nil
}

// https://dev.to/chigbeef_77/bool-int-but-stupid-in-go-3jb3
func boolToByte(b bool) byte {
	var i byte
	if b {
		i = 1
	} else {
		i = 0
	}

	return i
}
