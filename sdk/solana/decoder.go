package solana

import (
	// "strings"

	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/text"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

type Decoder struct{}

var _ sdk.Decoder = &Decoder{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d *Decoder) Decode(tx types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	var additionalFields AdditionalFields
	if len(tx.AdditionalFields) != 0 {
		err := json.Unmarshal(tx.AdditionalFields, &additionalFields)
		if err != nil {
			return &DecodedOperation{}, fmt.Errorf("failed to unmarshal additional fields: %w", err)
		}
	}

	return ParseFunctionCall(contractInterfaces, tx.Data, additionalFields.Accounts)
}

// ParseFunctionCall parses a full data payload (with function selector at the front of it) and a full contract ABI
// into a function name and an array of inputs.
func ParseFunctionCall(idlJSON string, data []byte, accounts []*solana.AccountMeta) (*DecodedOperation, error) {
	var idl IDL
	err := json.Unmarshal([]byte(idlJSON), &idl)
	if err != nil {
		return &DecodedOperation{}, fmt.Errorf("failed to unmarshal IDL: %w", err)
	}
	spew.Dump(idl)
	// decodeSystemTransfer(tx)

	// // Parse the ABI
	// parsedAbi, err := geth_abi.JSON(strings.NewReader(fullAbi))
	// if err != nil {
	// 	return &DecodedOperation{}, err
	// }
	//
	// // Extract the method from the data
	// method, err := parsedAbi.MethodById(data[:4])
	// if err != nil {
	// 	return &DecodedOperation{}, err
	// }
	//
	// // Unpack the data
	// inputs, err := method.Inputs.UnpackValues(data[4:])
	// if err != nil {
	// 	return &DecodedOperation{}, err
	// }
	//
	// // Get the keys of the inputs
	// methodKeys := make([]string, len(method.Inputs))
	// for i, input := range method.Inputs {
	// 	methodKeys[i] = input.Name
	// }
	//
	return &DecodedOperation{
		FunctionName: idl.Name,
		// InputKeys:    methodKeys,
		// InputArgs:    inputs,
	}, nil
}

func decodeSystemTransfer(tx *solana.Transaction) {
	// spew.Dump(tx)

	// Get (for example) the first instruction of this transaction
	// which we know is a `system` program instruction:
	i0 := tx.Message.Instructions[0]

	// Find the program address of this instruction:
	progKey, err := tx.ResolveProgramIDIndex(i0.ProgramIDIndex)
	if err != nil {
		panic(err)
	}

	// Find the accounts of this instruction:
	accounts, err := i0.ResolveInstructionAccounts(&tx.Message)
	if err != nil {
		panic(err)
	}

	// Feed the accounts and data to the system program parser
	// OR see below for alternative parsing when you DON'T know
	// what program the instruction is for / you don't have a parser.
	inst, err := system.DecodeInstruction(accounts, i0.Data)
	if err != nil {
		panic(err)
	}

	// inst.Impl contains the specific instruction type (in this case, `inst.Impl` is a `*system.Transfer`)
	spew.Dump(inst)
	if _, ok := inst.Impl.(*system.Transfer); !ok {
		panic("the instruction is not a *system.Transfer")
	}

	// OR
	{
		// There is a more general instruction decoder: `solana.DecodeInstruction`.
		// But before you can use `solana.DecodeInstruction`,
		// you must register a decoder for each program ID beforehand
		// by using `solana.RegisterInstructionDecoder` (all solana-go program clients do it automatically with the default program IDs).
		decodedInstruction, err := solana.DecodeInstruction(
			progKey,
			accounts,
			i0.Data,
		)
		if err != nil {
			panic(err)
		}
		// The returned `decodedInstruction` is the decoded instruction.
		spew.Dump(decodedInstruction)

		// decodedInstruction == inst
		if !reflect.DeepEqual(inst, decodedInstruction) {
			panic("they are NOT equal (this would never happen)")
		}

		// To register other (not yet registered decoders), you can add them with
		// `solana.RegisterInstructionDecoder` function.
	}

	{
		// pretty-print whole transaction:
		_, err := tx.EncodeTree(text.NewTreeEncoder(os.Stdout, text.Bold("TEST TRANSACTION")))
		if err != nil {
			panic(err)
		}
	}
}
