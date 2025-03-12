package aptos

import (
	"fmt"
	"strings"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/api"
)

func BuildSignSubmitAndWaitForTransaction(client aptos.AptosRpcClient, sender aptos.TransactionSigner, payload *aptos.TransactionPayload) (*api.UserTransaction, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload required")
	}
	submitResult, err := client.BuildSignAndSubmitTransaction(sender, *payload)
	if err != nil {
		return nil, fmt.Errorf("creating tx: %w", err)
	}
	data, err := client.WaitForTransaction(submitResult.Hash)
	if err != nil {
		return nil, fmt.Errorf("waiting for tx: %w", err)
	}
	if !data.Success {
		return nil, fmt.Errorf("transaction failed: %s", data.VmStatus)
	}

	return data, nil
}

func BuildTransactionPayload(function string, typeArgs, paramTypes []string, paramValues []any) (*aptos.TransactionPayload, error) {
	functionTokens := strings.Split(function, "::")
	if len(functionTokens) != 3 {
		return nil, fmt.Errorf("unexpected function name, expected 3 tokens, got %d", len(functionTokens))
	}
	if len(paramTypes) != len(paramValues) {
		return nil, fmt.Errorf("length of param types and param values do not match")
	}

	contractAccountAddress := aptos.AccountAddress{}
	if err := contractAccountAddress.ParseStringRelaxed(functionTokens[0]); err != nil {
		return nil, fmt.Errorf("failed to parse contract account address: %w", err)
	}
	moduleName := functionTokens[1]
	functionName := functionTokens[2]

	typeTags := make([]aptos.TypeTag, len(typeArgs))
	args := make([][]byte, len(paramTypes))

	for i, arg := range typeArgs {
		typeTag, err := CreateTypeTag(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type argument %q: %w", arg, err)
		}
		typeTags[i] = typeTag
	}
	for i := range paramTypes {
		typeName := paramTypes[i]
		typeValue := paramValues[i]

		typeTag, err := CreateTypeTag(typeName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type %q: %w", typeName, err)
		}

		bcsValue, err := CreateBcsValue(typeTag, typeValue)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize value #%v %q: %w", i, typeValue, err)
		}
		args[i] = bcsValue
	}

	return &aptos.TransactionPayload{Payload: &aptos.EntryFunction{
		Module: aptos.ModuleId{
			Address: contractAccountAddress,
			Name:    moduleName,
		},
		Function: functionName,
		ArgTypes: typeTags,
		Args:     args,
	}}, nil
}

func BuildViewPayload(function string, typeArgs, paramTypes []string, paramValues []any) (*aptos.ViewPayload, error) {
	// TODO This is just the same as above
	functionTokens := strings.Split(function, "::")
	if len(functionTokens) != 3 {
		return nil, fmt.Errorf("unexpected function name, expected 3 tokens, got %d", len(functionTokens))
	}
	if len(paramTypes) != len(paramValues) {
		return nil, fmt.Errorf("length of param types and param values do not match")
	}

	contractAccountAddress := aptos.AccountAddress{}
	if err := contractAccountAddress.ParseStringRelaxed(functionTokens[0]); err != nil {
		return nil, fmt.Errorf("failed to parse contract account address: %w", err)
	}
	moduleName := functionTokens[1]
	functionName := functionTokens[2]

	typeTags := make([]aptos.TypeTag, len(typeArgs))
	args := make([][]byte, len(paramTypes))

	for i, arg := range typeArgs {
		typeTag, err := CreateTypeTag(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type argument %q: %w", arg, err)
		}
		typeTags[i] = typeTag
	}
	for i := range paramTypes {
		typeName := paramTypes[i]
		typeValue := paramValues[i]

		typeTag, err := CreateTypeTag(typeName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type %q: %w", typeName, err)
		}

		bcsValue, err := CreateBcsValue(typeTag, typeValue)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize value #%v %q: %w", i, typeValue, err)
		}
		args[i] = bcsValue
	}

	return &aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contractAccountAddress,
			Name:    moduleName,
		},
		Function: functionName,
		ArgTypes: typeTags,
		Args:     args,
	}, nil
}
