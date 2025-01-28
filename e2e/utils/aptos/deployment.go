package aptos

import (
	"fmt"
	"strings"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
)

func replaceAddress(s string, old, new string) (string, error) {
	old, new = strings.TrimPrefix(old, "0x"), strings.TrimPrefix(new, "0x")
	repl := strings.ReplaceAll(
		strings.ToLower(s),
		strings.ToLower(old),
		strings.ToLower(new),
	)
	res, err := aptos.ParseHex(repl)
	if err != nil {
		return "", err
	}
	return aptos.BytesToHex(res), nil
}

func replaceAddresses(s string, replacements map[string]string) (string, error) {
	var err error
	for old, neww := range replacements {
		s, err = replaceAddress(s, old, neww)
		if err != nil {
			return "", err
		}
	}
	return s, nil
}

// ObjectCodeDeploymentPublish calls 0x1::object_code_deployment::publish
// https://github.com/aptos-labs/aptos-core/blob/main/aptos-move/framework/aptos-framework/doc/object_code_deployment.md#function-publish
func ObjectCodeDeploymentPublish(metadataHex string, bytecodeHex []string, addresses map[string]string) (*aptos.TransactionPayload, error) {
	metadataHex, err := replaceAddresses(metadataHex, addresses)
	if err != nil {
		return nil, err
	}
	for i, s := range bytecodeHex {
		// Modifies the slice, possibly copy first
		bytecodeHex[i], err = replaceAddresses(s, addresses)
		if err != nil {
			return nil, err
		}
	}
	metadata, err := aptos.ParseHex(metadataHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hex metadata: %w", err)
	}
	bytecode := make([][]byte, len(bytecodeHex))
	for i, hex := range bytecodeHex {
		bytecode[i], err = aptos.ParseHex(hex)
		if err != nil {
			return nil, fmt.Errorf("failed to parse hex bytecode: %w", err)
		}
	}
	return BuildTransactionPayload(
		"0x1::object_code_deployment::publish",
		nil,
		[]string{"vector<u8>", "vector<vector<u8>>"},
		[]any{metadata, bytecode},
	)
}

// CalculateNextObjectCodeDeploymentAddress calculates the address of the next named object that will be created
// when performing an object code deployment using 0x1::object_code_deployment::publish
// It uses 0x1::object::create_named_object with the seed being the sending addresses next sequence + a fixed domain separator
func CalculateNextObjectCodeDeploymentAddress(address aptos.AccountAddress, currSeq uint64) aptos.AccountAddress {
	sequenceBytes, _ := bcs.SerializeU64(currSeq + 1)
	domainSeparatorBytes, _ := bcs.SerializeBytes([]byte("aptos_framework::object_code_deployment"))
	seedBytes := append(domainSeparatorBytes, sequenceBytes...)
	return address.NamedObjectAddress(seedBytes)
}

func NextObjectCodeDeploymentAddress(client *aptos.NodeClient, account aptos.AccountAddress) (aptos.AccountAddress, error) {
	accountInfo, err := client.Account(account)
	if err != nil {
		return aptos.AccountAddress{}, fmt.Errorf("failed to get account info: %w", err)
	}
	sequence, err := accountInfo.SequenceNumber()
	if err != nil {
		return aptos.AccountAddress{}, fmt.Errorf("failed to get sequence number: %w", err)
	}
	return CalculateNextObjectCodeDeploymentAddress(account, sequence), nil
}
